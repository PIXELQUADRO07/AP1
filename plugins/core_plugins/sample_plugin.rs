// Esempio di plugin Rust per AP1 core.

pub fn initialize() {
    println!("AP1 core plugin inizializzato: sample_plugin");
}

pub fn process_packet(_data: &[u8]) {
    // Inserire qui la logica di gestione del pacchetto.
}
