use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::sync::{LazyLock, Mutex};
use vodozemac::olm::*;
use vodozemac::Curve25519PublicKey;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;

struct CryptoState {
    account: Account,
    sessions: HashMap<String, Box<Session>>,
}

static STATE: LazyLock<Mutex<Option<CryptoState>>> =
    LazyLock::new(|| Mutex::new(None));

// ─── Public Rust API ─────────────────────────────────────────

pub fn rust_generate_account() -> Result<String, String> {
    let account = Account::new();
    let id_keys = account.identity_keys();
    let bundle = serde_json::json!({
        "curve25519": id_keys.curve25519.to_base64(),
        "ed25519": id_keys.ed25519.to_base64(),
    });
    let mut guard = STATE.lock().unwrap();
    *guard = Some(CryptoState {
        account,
        sessions: HashMap::new(),
    });
    Ok(bundle.to_string())
}

pub fn rust_get_identity_keys() -> Result<String, String> {
    let guard = STATE.lock().unwrap();
    let state = guard.as_ref().ok_or("not initialized")?;
    let id_keys = state.account.identity_keys();
    let bundle = serde_json::json!({
        "curve25519": id_keys.curve25519.to_base64(),
        "ed25519": id_keys.ed25519.to_base64(),
    });
    Ok(bundle.to_string())
}

pub fn rust_generate_one_time_keys(count: usize) -> Result<String, String> {
    let mut guard = STATE.lock().unwrap();
    let state = guard.as_mut().ok_or("not initialized")?;
    state.account.generate_one_time_keys(count);
    let keys: Vec<String> = state
        .account
        .one_time_keys()
        .iter()
        .map(|(_, k)| k.to_base64())
        .collect();
    state.account.mark_keys_as_published();
    serde_json::to_string(&keys).map_err(|e| e.to_string())
}

pub fn rust_sign_message(message: &str) -> Result<String, String> {
    let guard = STATE.lock().unwrap();
    let state = guard.as_ref().ok_or("not initialized")?;
    Ok(state.account.sign(message).to_base64())
}

pub fn rust_create_outbound_session(
    their_identity_key: &str,
    their_one_time_key: &str,
) -> Result<String, String> {
    let mut guard = STATE.lock().unwrap();
    let state = guard.as_mut().ok_or("not initialized")?;
    let their_id = Curve25519PublicKey::from_base64(their_identity_key)
        .map_err(|e| format!("bad identity key: {}", e))?;
    let their_otk = Curve25519PublicKey::from_base64(their_one_time_key)
        .map_err(|e| format!("bad one-time key: {}", e))?;
    let session = state
        .account
        .create_outbound_session(SessionConfig::default(), their_id, their_otk)
        .map_err(|e| format!("session creation failed: {}", e))?;
    let session_id = session.session_id().to_owned();
    let info = serde_json::json!({ "session_id": session_id });
    state.sessions.insert(session.session_id().to_owned(), Box::new(session));
    Ok(info.to_string())
}

pub fn rust_encrypt_message(session_id: &str, plaintext: &str) -> Result<String, String> {
    let mut guard = STATE.lock().unwrap();
    let state = guard.as_mut().ok_or("not initialized")?;
    let session = state
        .sessions
        .get_mut(session_id)
        .ok_or("session not found")?;
    let msg = session
        .encrypt(plaintext)
        .map_err(|e| format!("encrypt failed: {}", e))?;
    let (msg_type, ciphertext) = msg.to_parts();
    let payload = serde_json::json!({
        "ciphertext": base64::Engine::encode(&base64::engine::general_purpose::STANDARD, ciphertext),
        "message_type": msg_type,
    });
    Ok(payload.to_string())
}

pub fn rust_create_inbound_session(
    their_identity_key: &str,
    ciphertext_b64: &str,
) -> Result<String, String> {
    let mut guard = STATE.lock().unwrap();
    let state = guard.as_mut().ok_or("not initialized")?;
    let ciphertext = base64::Engine::decode(
        &base64::engine::general_purpose::STANDARD,
        ciphertext_b64,
    )
    .map_err(|e| format!("base64 decode: {}", e))?;
    let pre_key_msg = PreKeyMessage::from_bytes(&ciphertext)
        .map_err(|e| format!("bad pre-key message: {}", e))?;
    let their_key = Curve25519PublicKey::from_base64(their_identity_key)
        .map_err(|e| format!("bad identity key: {}", e))?;
    let result = state
        .account
        .create_inbound_session(SessionConfig::default(), their_key, &pre_key_msg)
        .map_err(|e| format!("inbound session: {}", e))?;
    let session_id = result.session.session_id().to_owned();
    state.sessions.insert(session_id, Box::new(result.session));
    Ok(String::from_utf8_lossy(&result.plaintext).to_string())
}

pub fn rust_decrypt_message(session_id: &str, ciphertext_b64: &str, message_type: usize) -> Result<String, String> {
    let mut guard = STATE.lock().unwrap();
    let state = guard.as_mut().ok_or("not initialized")?;
    let session = state
        .sessions
        .get_mut(session_id)
        .ok_or("session not found")?;
    let ciphertext = base64::Engine::decode(
        &base64::engine::general_purpose::STANDARD,
        ciphertext_b64,
    )
    .map_err(|e| format!("base64 decode: {}", e))?;
    let msg = OlmMessage::from_parts(message_type, &ciphertext)
        .map_err(|e| format!("bad olm message: {}", e))?;
    let plaintext = session
        .decrypt(&msg)
        .map_err(|e| format!("decrypt failed: {}", e))?;
    Ok(String::from_utf8_lossy(&plaintext).to_string())
}

pub fn rust_get_fingerprint(their_identity_key: &str) -> Result<String, String> {
    let guard = STATE.lock().unwrap();
    let state = guard.as_ref().ok_or("not initialized")?;
    let our_key = state.account.identity_keys().curve25519.to_base64();
    let mut hasher = Sha256::new();
    hasher.update(our_key.as_bytes());
    hasher.update(their_identity_key.as_bytes());
    let hash = hasher.finalize();
    let hex: String = hash.iter().take(16).map(|b| format!("{:02x}", b)).collect();
    let formatted = hex
        .as_bytes()
        .chunks(4)
        .map(|c| std::str::from_utf8(c).unwrap_or(""))
        .collect::<Vec<&str>>()
        .join(" ");
    Ok(formatted)
}

pub fn rust_pickle_account() -> Result<String, String> {
    let guard = STATE.lock().unwrap();
    let state = guard.as_ref().ok_or("not initialized")?;
    let pickle = state.account.pickle();
    serde_json::to_string(&pickle).map_err(|e| e.to_string())
}

pub fn rust_unpickle_account(pickle_json: &str) -> Result<String, String> {
    let pickle: AccountPickle =
        serde_json::from_str(pickle_json).map_err(|e| format!("bad pickle: {}", e))?;
    let account = Account::from_pickle(pickle);
    let id_keys = account.identity_keys();
    let bundle = serde_json::json!({
        "curve25519": id_keys.curve25519.to_base64(),
        "ed25519": id_keys.ed25519.to_base64(),
    });
    let mut guard = STATE.lock().unwrap();
    *guard = Some(CryptoState {
        account,
        sessions: HashMap::new(),
    });
    Ok(bundle.to_string())
}

pub fn rust_pickle_session(session_id: &str) -> Result<String, String> {
    let guard = STATE.lock().unwrap();
    let state = guard.as_ref().ok_or("not initialized")?;
    let session = state
        .sessions
        .get(session_id)
        .ok_or("session not found")?;
    let pickle = session.pickle();
    serde_json::to_string(&pickle).map_err(|e| e.to_string())
}

pub fn rust_unpickle_session(session_id: &str, pickle_json: &str) -> Result<String, String> {
    let pickle: SessionPickle =
        serde_json::from_str(pickle_json).map_err(|e| format!("bad session pickle: {}", e))?;
    let session = Session::from_pickle(pickle);
    let mut guard = STATE.lock().unwrap();
    let state = guard.as_mut().ok_or("not initialized")?;
    let sid = session.session_id().to_owned();
    state.sessions.insert(sid.clone(), Box::new(session));
    if session_id != sid {
        state.sessions.remove(session_id);
    }
    Ok(serde_json::json!({"session_id": sid}).to_string())
}

// ─── C FFI Layer ─────────────────────────────────────────────

fn c_result(result: Result<String, String>) -> *mut c_char {
    let output = match result {
        Ok(val) => format!("OK:{}", val),
        Err(err) => format!("ERR:{}", err),
    };
    CString::new(output).unwrap_or_default().into_raw()
}

fn c_string_to_rust(ptr: *const c_char) -> Result<String, String> {
    if ptr.is_null() {
        return Err("null pointer".to_string());
    }
    unsafe { CStr::from_ptr(ptr) }
        .to_str()
        .map(|s| s.to_owned())
        .map_err(|e| format!("invalid UTF-8: {}", e))
}

#[no_mangle]
pub extern "C" fn yup_generate_account() -> *mut c_char {
    c_result(rust_generate_account())
}

#[no_mangle]
pub extern "C" fn yup_get_identity_keys() -> *mut c_char {
    c_result(rust_get_identity_keys())
}

#[no_mangle]
pub extern "C" fn yup_generate_one_time_keys(count: usize) -> *mut c_char {
    c_result(rust_generate_one_time_keys(count))
}

#[no_mangle]
pub extern "C" fn yup_sign_message(msg: *const c_char) -> *mut c_char {
    let msg = c_string_to_rust(msg).unwrap_or_default();
    c_result(rust_sign_message(&msg))
}

#[no_mangle]
pub extern "C" fn yup_create_outbound_session(
    their_identity_key: *const c_char,
    their_one_time_key: *const c_char,
) -> *mut c_char {
    let id_key = c_string_to_rust(their_identity_key).unwrap_or_default();
    let otk = c_string_to_rust(their_one_time_key).unwrap_or_default();
    c_result(rust_create_outbound_session(&id_key, &otk))
}

#[no_mangle]
pub extern "C" fn yup_encrypt_message(session_id: *const c_char, plaintext: *const c_char) -> *mut c_char {
    let sid = c_string_to_rust(session_id).unwrap_or_default();
    let pt = c_string_to_rust(plaintext).unwrap_or_default();
    c_result(rust_encrypt_message(&sid, &pt))
}

#[no_mangle]
pub extern "C" fn yup_create_inbound_session(
    their_identity_key: *const c_char,
    ciphertext_b64: *const c_char,
) -> *mut c_char {
    let id_key = c_string_to_rust(their_identity_key).unwrap_or_default();
    let ct = c_string_to_rust(ciphertext_b64).unwrap_or_default();
    c_result(rust_create_inbound_session(&id_key, &ct))
}

#[no_mangle]
pub extern "C" fn yup_decrypt_message(
    session_id: *const c_char,
    ciphertext_b64: *const c_char,
    message_type: usize,
) -> *mut c_char {
    let sid = c_string_to_rust(session_id).unwrap_or_default();
    let ct = c_string_to_rust(ciphertext_b64).unwrap_or_default();
    c_result(rust_decrypt_message(&sid, &ct, message_type))
}

#[no_mangle]
pub extern "C" fn yup_get_fingerprint(their_identity_key: *const c_char) -> *mut c_char {
    let id_key = c_string_to_rust(their_identity_key).unwrap_or_default();
    c_result(rust_get_fingerprint(&id_key))
}

#[no_mangle]
pub extern "C" fn yup_pickle_account() -> *mut c_char {
    c_result(rust_pickle_account())
}

#[no_mangle]
pub extern "C" fn yup_unpickle_account(pickle: *const c_char) -> *mut c_char {
    let pickle = c_string_to_rust(pickle).unwrap_or_default();
    c_result(rust_unpickle_account(&pickle))
}

#[no_mangle]
pub extern "C" fn yup_pickle_session(session_id: *const c_char) -> *mut c_char {
    let sid = c_string_to_rust(session_id).unwrap_or_default();
    c_result(rust_pickle_session(&sid))
}

#[no_mangle]
pub extern "C" fn yup_unpickle_session(
    session_id: *const c_char,
    pickle: *const c_char,
) -> *mut c_char {
    let sid = c_string_to_rust(session_id).unwrap_or_default();
    let p = c_string_to_rust(pickle).unwrap_or_default();
    c_result(rust_unpickle_session(&sid, &p))
}

#[no_mangle]
pub extern "C" fn yup_free_string(s: *mut c_char) {
    if !s.is_null() {
        unsafe { drop(CString::from_raw(s)); }
    }
}
