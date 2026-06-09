use pnet::datalink::{self, NetworkInterface};
use pnet::datalink::Channel::Ethernet;
use pnet::packet::ethernet::EthernetPacket;
use pnet::packet::ipv4::Ipv4Packet;
use pnet::packet::tcp::TcpPacket;
use pnet::packet::udp::UdpPacket;
use pnet::packet::Packet;
use std::thread;
use tracing::{info, error};
use crate::database;

pub fn start_sniffer(interface_name: String) {
    thread::spawn(move || {
        let interfaces = datalink::interfaces();
        let interface = interfaces
            .into_iter()
            .find(|iface: &NetworkInterface| iface.name == interface_name)
            .expect("Failed to find network interface");

        let (_, mut rx) = match datalink::channel(&interface, Default::default()) {
            Ok(Ethernet(tx, rx)) => (tx, rx),
            Ok(_) => panic!("Unhandled channel type"),
            Err(e) => panic!("Failed to create datalink channel: {}", e),
        };

        info!("Advanced sniffer started on interface: {}", interface_name);

        loop {
            match rx.next() {
                Ok(packet) => {
                    let eth_packet = EthernetPacket::new(packet).unwrap();
                    handle_ethernet_packet(&eth_packet);
                }
                Err(e) => {
                    error!("Sniffer error: {}", e);
                }
            }
        }
    });
}

fn handle_ethernet_packet(eth_packet: &EthernetPacket) {
    if let Some(ip_packet) = Ipv4Packet::new(eth_packet.payload()) {
        match ip_packet.get_next_level_protocol() {
            pnet::packet::ip::IpNextHeaderProtocols::Tcp => {
                if let Some(tcp_packet) = TcpPacket::new(ip_packet.payload()) {
                    handle_tcp_packet(&ip_packet, &tcp_packet);
                }
            }
            pnet::packet::ip::IpNextHeaderProtocols::Udp => {
                if let Some(udp_packet) = UdpPacket::new(ip_packet.payload()) {
                    handle_udp_packet(&ip_packet, &udp_packet);
                }
            }
            _ => {}
        }
    }
}

fn handle_tcp_packet(ip_packet: &Ipv4Packet, tcp_packet: &TcpPacket) {
    let source = format!("{}:{}", ip_packet.get_source(), tcp_packet.get_source());
    let destination = format!("{}:{}", ip_packet.get_destination(), tcp_packet.get_destination());

    // HTTP detection (Port 80)
    if tcp_packet.get_destination() == 80 || tcp_packet.get_source() == 80 {
        let payload = tcp_packet.payload();
        if !payload.is_empty() {
            let data = String::from_utf8_lossy(payload);
            if data.contains("GET") || data.contains("POST") || data.contains("Host:") {
                let first_line = data.lines().next().unwrap_or("");
                database::log_traffic(&source, &destination, "HTTP", first_line);
            }
        }
    }
}

fn handle_udp_packet(ip_packet: &Ipv4Packet, udp_packet: &UdpPacket) {
    let source = format!("{}:{}", ip_packet.get_source(), udp_packet.get_source());
    let destination = format!("{}:{}", ip_packet.get_destination(), udp_packet.get_destination());

    // DNS detection (Port 53)
    if udp_packet.get_destination() == 53 || udp_packet.get_source() == 53 {
        let payload = udp_packet.payload();
        if payload.len() > 12 {
            // DNS header is 12 bytes.
            // Simple check for QNAME: starts at 13th byte (index 12)
            let qname = parse_dns_qname(&payload[12..]);
            if !qname.is_empty() {
                database::log_traffic(&source, &destination, "DNS", &format!("Query: {}", qname));
            } else {
                database::log_traffic(&source, &destination, "DNS", "Query detected");
            }
        }
    }
}

fn parse_dns_qname(payload: &[u8]) -> String {
    let mut qname = String::new();
    let mut i = 0;
    while i < payload.len() {
        let len = payload[i] as usize;
        if len == 0 { break; }
        if i + 1 + len > payload.len() { break; }
        if !qname.is_empty() { qname.push('.'); }
        qname.push_str(&String::from_utf8_lossy(&payload[i+1..i+1+len]));
        i += 1 + len;
    }
    qname
}
