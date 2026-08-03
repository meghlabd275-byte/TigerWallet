use crate::database::Database;
use crate::error::Error;
use serde::{Deserialize, Serialize};

pub struct $(echo user_service | sed 's/_/\L&/g' | sed 's/^\(.\)/\U\1/')Service { db: Database }

impl $(echo user_service | sed 's/_/\L&/g' | sed 's/^\(.\)/\U\1/')Service {
    pub fn new(db: Database) -> Self { Self { db } }
}
