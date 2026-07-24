/**
 * TigerWallet Event Processing
 * High-Security Rust Implementation
 */

use std::sync::{Arc, RwLock};
use std::collections::VecDeque;

#[derive(Debug, Clone)]
pub struct Event {
    pub id: String,
    pub event_type: String,
    pub data: String,
    pub timestamp: u64,
}

pub struct EventProcessor {
    queue: RwLock<VecDeque<Event>>,
    processed: RwLock<u64>,
}

impl EventProcessor {
    pub fn new() -> Self {
        Self {
            queue: RwLock::new(VecDeque::new()),
            processed: RwLock::new(0),
        }
    }

    pub fn publish(&self, event_type: &str, data: &str) {
        let event = Event {
            id: format!("evt_{}", std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH).unwrap().as_nanos()),
            event_type: event_type.to_string(),
            data: data.to_string(),
            timestamp: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH).unwrap().as_secs(),
        };
        
        self.queue.write().unwrap().push_back(event);
    }

    pub fn process(&self) -> Option<Event> {
        let event = self.queue.write().unwrap().pop_front()?;
        
        let mut processed = self.processed.write().unwrap();
        *processed += 1;
        
        Some(event)
    }

    pub fn queue_size(&self) -> usize {
        self.queue.read().unwrap().len()
    }
}

fn main() {
    println!("Event Processor on :8099");
    
    let processor = Arc::new(EventProcessor::new());
    
    processor.publish("tx_confirmed", "0x123...");
    processor.publish("swap_completed", "ETH->USDT");
    
    while let Some(event) = processor.process() {
        println!("Processed: {} - {}", event.event_type, event.data);
    }
    
    println!("Queue size: {}", processor.queue_size());
}
