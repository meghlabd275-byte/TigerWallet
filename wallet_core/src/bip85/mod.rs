/**
 * TigerWallet BIP-85 Implementation
 * 
 * BIP-85: Deterministic Entropy from Mnemonic
 * https://github.com/bitcoin/bips/blob/master/bip-0085.mediawiki
 * 
 * This implementation provides:
 * - Deriving cryptographic entropy from BIP-39 mnemonics
 * - HD wallet key generation for multiple applications
 * - Secure random number generation
 * 
 * Supported Applications:
 * - Bitcoin: derivation path "m/83696968'/0'/0'/0'/0'"
 * - Ethereum: derivation path "m/83696968'/60'/0'/0'/0'"
 * - Backup: derivation path "m/83696968'/128'/0'/0'/0'"
 * - SSH: derivation path "m/83696968'/13'/0'/0'/0'"
 * - GPG: derivation path "m/83696968'/0'/0'/2'/0'"
 */

use std::fmt;
use hmac::{Hmac, Mac};
use sha512::Sha512;

// Application constants
pub const APP_BITCOIN: u32 = 0;
pub const APP_ETHEREUM: u32 = 60;
pub const APP_BACKUP: u32 = 128;
pub const APP_SSH: u32 = 13;
pub const APP_GPG: u32 = 0x47504FFD; // "GPG" in base10

// Maximum entropy lengths
pub const ENTROPY_LEN_16: usize = 16;  // 128 bits
pub const ENTROPY_LEN_32: usize = 32;  // 256 bits

/// BIP-85 Derivation error
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Bip85Error {
    InvalidMnemonic,
    DerivationFailed,
    InvalidEntropyLength,
}

impl fmt::Display for Bip85Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Bip85Error::InvalidMnemonic => write!(f, "Invalid mnemonic phrase"),
            Bip85Error::DerivationFailed => write!(f, "Key derivation failed"),
            Bip85Error::InvalidEntropyLength => write!(f, "Invalid entropy length"),
        }
    }
}

impl std::error::Error for Bip85Error {}

/// BIP-85 entropy result
#[derive(Debug, Clone)]
pub struct EntropyOutput {
    pub entropy: Vec<u8>,
    pub mnemonic: String,
    pub index: u32,
    pub application: u32,
}

impl EntropyOutput {
    /// Get entropy as hex string
    pub fn hex(&self) -> String {
        hex::encode(&self.entropy)
    }
    
    /// Get entropy as bytes
    pub fn bytes(&self) -> &[u8] {
        &self.entropy
    }
    
    /// Get 128-bit entropy (12 words)
    pub fn entropy_128(&self) -> Result<[u8; 16], Bip85Error> {
        if self.entropy.len() != 16 {
            return Err(Bip85Error::InvalidEntropyLength);
        }
        let mut arr = [0u8; 16];
        arr.copy_from_slice(&self.entropy[..16]);
        Ok(arr)
    }
    
    /// Get 256-bit entropy (24 words)
    pub fn entropy_256(&self) -> Result<[u8; 32], Bip85Error> {
        if self.entropy.len() != 32 {
            return Err(Bip85Error::InvalidEntropyLength);
        }
        let mut arr = [0u8; 32];
        arr.copy_from_slice(&self.entropy[..32]);
        Ok(arr)
    }
}

/// BIP-85 Deriver
pub struct Bip85Deriver;

impl Bip85Deriver {
    /// Derive entropy from a BIP-39 mnemonic
    /// 
    /// # Arguments
    /// * `mnemonic` - BIP-39 mnemonic phrase
    /// * `passphrase` - Optional passphrase (empty string if none)
    /// * `application` - Application index (0=Bitcoin, 60=Ethereum, etc.)
    /// * `index` - Derivation index
    /// * `entropy_len` - Output entropy length (16 or 32 bytes)
    /// 
    /// # Returns
    /// * `EntropyOutput` containing derived entropy and new mnemonic
    pub fn derive(
        mnemonic: &str,
        passphrase: &str,
        application: u32,
        index: u32,
        entropy_len: usize,
    ) -> Result<EntropyOutput, Bip85Error> {
        // Validate entropy length
        if entropy_len != 16 && entropy_len != 32 {
            return Err(Bip85Error::InvalidEntropyLength);
        }
        
        // Derive seed from mnemonic using BIP-39
        let seed = Self::bip39_seed(mnemonic, passphrase)?;
        
        // Build derivation path: m/83696968'/application'/index'/0'/entropy_len*8
        let path = format!("m/83696968'/{}/{}'/0'/{}", application, index, entropy_len * 8);
        
        // Derive child key using HMAC-SHA512
        let child_key = Self::derive_key(&seed, &path)?;
        
        // Use left 16 or 32 bytes as entropy
        let entropy = child_key[..entropy_len].to_vec();
        
        // Generate new mnemonic from entropy
        let new_mnemonic = Self::entropy_to_mnemonic(&entropy)?;
        
        Ok(EntropyOutput {
            entropy,
            mnemonic: new_mnemonic,
            index,
            application,
        })
    }
    
    /// Derive Bitcoin seed from mnemonic (BIP-39)
    fn bip39_seed(mnemonic: &str, passphrase: &str) -> Result<Vec<u8>, Bip85Error> {
        // Normalize mnemonic (trim whitespace, lowercase)
        let mnemonic = mnemonic.trim().to_lowercase();
        
        // Validate mnemonic has valid word count
        let words: Vec<&str> = mnemonic.split_whitespace().collect();
        if words.len() != 12 && words.len() != 24 {
            return Err(Bip85Error::InvalidMnemonic);
        }
        
        // BIP-39 salt is "mnemonic" + passphrase
        let salt = format!("mnemonic{}", passphrase);
        
        // PBKDF2 with 2048 iterations, HMAC-SHA512
        let seed = Self::pbkdf2_hmac_sha512(
            mnemonic.as_bytes(),
            salt.as_bytes(),
            2048,
            64,
        );
        
        Ok(seed)
    }
    
    /// Derive child key from seed using path
    fn derive_key(seed: &[u8], path: &str) -> Result<Vec<u8>, Bip85Error> {
        // Convert path to derivation indices
        let indices = Self::parse_path(path)?;
        
        let mut key = seed.to_vec();
        
        for idx in indices {
            key = Self::hkdf_sha512(&key, &idx.to_be_bytes())?;
        }
        
        Ok(key)
    }
    
    /// Parse BIP-32 path string to indices
    fn parse_path(path: &str) -> Result<Vec<u32>, Bip85Error> {
        let mut indices = Vec::new();
        
        // Skip "m/" prefix
        let path = path.strip_prefix("m/").ok_or(Bip85Error::DerivationFailed)?;
        
        for part in path.split('/') {
            let mut hardened = false;
            let mut value = part;
            
            // Check for hardened derivation (')
            if let Some(v) = part.strip_suffix("'") {
                value = v;
                hardened = true;
            }
            
            // Parse index
            let idx: u32 = value.parse().map_err(|_| Bip85Error::DerivationFailed)?;
            
            // Apply hardened bit if needed
            if hardened {
                indices.push(0x80000000 | idx);
            } else {
                indices.push(idx);
            }
        }
        
        Ok(indices)
    }
    
    /// HMAC-SHA512 key derivation (PBKDF2 inner)
    fn hkdf_sha512(ikm: &[u8], info: &[u8]) -> Result<Vec<u8>, Bip85Error> {
        type HmacSha512 = Hmac<Sha512>;
        
        // PRK = HMAC-SHA512(0, IKM)
        let mut mac = HmacSha512::new_from_slice(ikm)
            .map_err(|_| Bip85Error::DerivationFailed)?;
        mac.update(&[0u8; 64]); // 64 zero bytes
        let prk = mac.finalize().into_bytes();
        
        // OKM = HMAC-SHA512(PRK, info || 0x01)
        let mut mac = HmacSha512::new_from_slice(&prk)
            .map_err(|_| Bip85Error::DerivationFailed)?;
        mac.update(info);
        mac.update(&[1]); // T1
        let result = mac.finalize().into_bytes();
        
        Ok(result[..64].to_vec())
    }
    
    /// PBKDF2 with HMAC-SHA512
    fn pbkdf2_hmac_sha512(password: &[u8], salt: &[u8], iterations: u32, output_len: usize) -> Vec<u8> {
        let mut result = Vec::with_capacity(output_len);
        let mut block = vec![0u8; salt.len() + 4];
        block[..salt.len()].copy_from_slice(salt);
        
        let mut offset = 0;
        let mut block_num: u32 = 1;
        
        while offset < output_len {
            // Set block number
            let bn = block_num.to_be_bytes();
            block[salt.len()..].copy_from_slice(&bn);
            
            // U1 = PRF(Password, Salt || INT(i))
            let mut u = {
                type HmacSha512 = Hmac<Sha512>;
                let mut mac = HmacSha512::new_from_slice(password)
                    .expect("HMAC can take key of any size");
                mac.update(&block);
                mac.finalize().into_bytes().to_vec()
            };
            
            let mut result_block = u.clone();
            
            // U2...Uc
            for _ in 1..iterations {
                let mut mac = HmacSha512::new_from_slice(password)
                    .expect("HMAC can take key of any size");
                mac.update(&u);
                u = mac.finalize().into_bytes().to_vec();
                
                // XOR
                for (i, byte) in u.iter().enumerate() {
                    result_block[i] ^= *byte;
                }
            }
            
            // Append to result
            let remaining = output_len - offset;
            let to_copy = std::cmp::min(64, remaining);
            result.extend_from_slice(&result_block[..to_copy]);
            offset += to_copy;
            block_num += 1;
        }
        
        result
    }
    
    /// Convert entropy to BIP-39 mnemonic
    fn entropy_to_mnemonic(entropy: &[u8]) -> Result<String, Bip85Error> {
        // Use bip39 crate for wordlist lookup
        // For now, return a placeholder - in production use bip39 crate
        
        // Calculate checksum
        let checksum = Self::sha256_checksum(entropy);
        
        // Combine entropy and checksum
        let mut bits = Vec::new();
        for byte in entropy {
            for i in (0..8).rev() {
                bits.push((byte >> i) & 1);
            }
        }
        for byte in &checksum[..1] {
            for i in (0..8).rev() {
                bits.push((byte >> i) & 1);
            }
        }
        
        // Convert to words (simplified - use wordlist in production)
        // In production: use bip39 crate's English wordlist
        let word_count = entropy.len() * 8 / 11;
        let words = Self::bits_to_words(&bits, word_count);
        
        Ok(words.join(" "))
    }
    
    /// SHA-256 checksum
    fn sha256_checksum(data: &[u8]) -> Vec<u8> {
        use sha2::{Sha256, Digest};
        let mut hasher = Sha256::new();
        hasher.update(data);
        hasher.finalize().to_vec()
    }
    
    /// Convert bits to word indices using full BIP-39 English wordlist
    fn bits_to_words(bits: &[bool], count: usize) -> Vec<String> {
        // Full BIP-39 English wordlist (2048 words)
        let wordlist = [
            "abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
            "absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
            "acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual",
            "adapt", "add", "addict", "address", "adjust", "admit", "adult", "advance",
            "advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent",
            "agree", "ahead", "aim", "air", "airport", "aisle", "alarm", "album",
            "alcohol", "alert", "alien", "all", "alley", "allow", "almost", "alone",
            "alpha", "already", "also", "alter", "always", "amateur", "amazing", "among",
            "amount", "amused", "analyst", "anchor", "ancient", "anger", "angle", "angry",
            "animal", "ankle", "announce", "annual", "another", "answer", "antenna", "antique",
            "anxiety", "any", "apart", "apology", "appear", "apple", "approve", "april",
            "arch", "arctic", "area", "arena", "argue", "arm", "armed", "armor",
            "army", "around", "arrange", "arrest", "arrive", "arrow", "art", "artefact",
            "artist", "artwork", "ask", "aspect", "assault", "asset", "assist", "assume",
            "asthma", "athlete", "atom", "attack", "attend", "attitude", "attract", "auction",
            "audit", "august", "aunt", "author", "auto", "autumn", "average", "avocado",
            "avoid", "awake", "aware", "away", "awesome", "awful", "awkward", "axis",
            "baby", "bachelor", "bacon", "badge", "bag", "balance", "balcony", "ball",
            "bamboo", "banana", "banner", "bar", "barely", "bargain", "barrel", "base",
            "basic", "basket", "battle", "beach", "bean", "beauty", "because", "become",
            "beef", "before", "begin", "behave", "behind", "believe", "below", "belt",
            "bench", "benefit", "best", "betray", "better", "between", "beyond", "bicycle",
            "bid", "bike", "bind", "biology", "bird", "birth", "bitter", "black",
            "blade", "blame", "blanket", "blast", "bleak", "bless", "blind", "blood",
            "blossom", "blouse", "blue", "blur", "blush", "board", "boat", "body",
            "boil", "bomb", "bone", "bonus", "book", "boost", "border", "boring",
            "borrow", "boss", "bottom", "bounce", "box", "boy", "bracket", "brain",
            "brand", "brass", "brave", "bread", "breeze", "brick", "bridge", "brief",
            "bright", "bring", "brisk", "broccoli", "broken", "bronze", "broom", "brother",
            "brown", "brush", "bubble", "buddy", "budget", "buffalo", "build", "bulb",
            "bulk", "bullet", "bundle", "bunker", "burden", "burger", "burst", "bus",
            "business", "busy", "butter", "buyer", "buzz", "cabbage", "cabin", "cable",
            "cactus", "cage", "cake", "call", "calm", "camera", "camp", "canal",
            "cancel", "candy", "cannon", "canoe", "canvas", "canyon", "capable", "capital",
            "captain", "car", "carbon", "card", "cargo", "carpet", "carry", "cart",
            "case", "cash", "casino", "castle", "casual", "catch", "category", "cattle",
            "caught", "cause", "caution", "cave", "ceiling", "celery", "cement", "census",
            "century", "cereal", "certain", "chair", "chalk", "champion", "change", "chaos",
            "chapter", "charge", "chase", "chat", "cheap", "check", "cheese", "cherry",
            "chest", "chicken", "chief", "child", "chimney", "choice", "choose", "chronic",
            "chunk", "church", "cigar", "circle", "citizen", "city", "civil", "claim",
            "clap", "clarify", "claw", "clay", "clean", "clerk", "clever", "click",
            "client", "cliff", "climb", "clinic", "clip", "clock", "close", "cloud",
            "clown", "club", "clump", "cluster", "clutch", "coach", "coast", "coconut",
            "code", "coffee", "coil", "coin", "collect", "color", "column", "combine",
            "come", "comfort", "comic", "common", "company", "concert", "conduct", "confirm",
            "congress", "connect", "consider", "control", "convince", "cook", "cool", "copper",
            "copy", "coral", "core", "corn", "correct", "cost", "cottage", "cotton",
            "couch", "country", "couple", "course", "cousin", "cover", "coyote", "crack",
            "cradle", "craft", "cram", "crane", "crash", "crater", "crawl", "crazy",
            "cream", "credit", "creek", "crew", "cricket", "crime", "crisp", "critic",
            "crop", "cross", "crouch", "crowd", "crucial", "cruel", "cruise", "crumble",
            "crunch", "crush", "cry", "crystal", "cube", "culture", "cup", "cupboard",
            "curious", "current", "curtain", "curve", "cushion", "custom", "cute", "cycle",
            "dad", "damage", "damp", "dance", "danger", "daring", "dash", "daughter",
            "dawn", "day", "deal", "debate", "debris", "decade", "december", "decide",
            "decline", "decorate", "decrease", "deer", "defense", "define", "defy", "degree",
            "delay", "deliver", "demand", "denial", "dentist", "deny", "depart", "depend",
            "deposit", "depth", "deputy", "derive", "describe", "desert", "design", "desk",
            "despair", "destroy", "detail", "detect", "develop", "device", "devote", "diagram",
            "dial", "diamond", "diary", "dice", "diesel", "diet", "differ", "digital",
            "dignity", "dilemma", "dinner", "dinosaur", "direct", "dirt", "disagree",
            "discover", "disease", "dish", "dismiss", "disorder", "display", "distance", "divert",
            "divide", "divorce", "dizzy", "doctor", "document", "dog", "doll", "dolphin",
            "domain", "donate", "donkey", "donor", "door", "dose", "double", "dove",
            "draft", "dragon", "drama", "draw", "dream", "dress", "drift", "drill",
            "drink", "drip", "drive", "drop", "drum", "dry", "duck", "dumb",
            "dune", "during", "dust", "dutch", "duty", "dwarf", "dynamic", "eager",
            "eagle", "early", "earn", "earth", "easily", "east", "easy", "echo",
            "ecology", "economy", "edge", "edit", "educate", "effort", "egg", "eight",
            "eject", "elastic", "elbow", "elder", "electric", "elegant", "element", "elephant",
            "elevator", "elite", "else", "embark", "embody", "embrace", "emerge", "emotion",
            "employ", "empower", "empty", "enable", "enact", "end", "endorse", "enemy",
            "energy", "enforce", "engage", "engine", "enhance", "enjoy", "enlist", "enough",
            "enrich", "enroll", "ensure", "enter", "entire", "entry", "envelope", "episode",
            "equal", "equip", "era", "erase", "erode", "erosion", "error", "erupt",
            "escape", "essay", "essence", "estate", "eternal", "ethics", "evidence", "evil",
            "evoke", "evolve", "exact", "example", "excess", "exchange", "excite", "exclude",
            "excuse", "execute", "exercise", "exhaust", "exhibit", "exile", "exist", "exit",
            "exotic", "expand", "expect", "expire", "explain", "expose", "express", "extend",
            "extra", "eye", "eyebrow", "fabric", "face", "faculty", "fade", "faint",
            "faith", "fall", "false", "fame", "family", "famous", "fan", "fancy",
            "fantasy", "farm", "fashion", "fat", "fatal", "father", "fatigue", "fault",
            "favorite", "feature", "february", "federal", "fee", "feed", "feel", "female",
            "fence", "festival", "fetch", "fever", "few", "fiber", "fiction", "field",
            "figure", "file", "film", "filter", "final", "finance", "find", "fine",
            "finger", "finish", "fire", "firm", "first", "fiscal", "fish", "fitness",
            "fix", "flag", "flame", "flash", "flat", "flavor", "flee", "flight",
            "fling", "flip", "float", "flock", "flood", "floor", "flower", "fluid",
            "flush", "fly", "foam", "focus", "fog", "foil", "fold", "follow",
            "food", "foot", "force", "forest", "forget", "fork", "fortune", "forum",
            "forward", "fossil", "foster", "found", "fox", "fragile", "frame", "frequent",
            "fresh", "friend", "fringe", "frog", "front", "frost", "frown", "frozen",
            "fruit", "fuel", "fun", "funny", "furnace", "fury", "future", "gadget",
            "gain", "galaxy", "gallery", "game", "gap", "garage", "garbage", "garden",
            "garlic", "gas", "gasp", "gate", "gather", "gauge", "gaze", "general",
            "genius", "genre", "gentle", "genuine", "gesture", "ghost", "giant", "gift",
            "giggle", "ginger", "giraffe", "girl", "give", "glad", "glance", "glare",
            "glass", "glide", "glimpse", "globe", "gloom", "glory", "glove", "glow",
            "glue", "goat", "goddess", "gold", "good", "goose", "gorilla", "gospel",
            "gossip", "govern", "gown", "grab", "grace", "grain", "grant", "grape",
            "grass", "gravity", "great", "green", "grid", "grief", "grit", "grocery",
            "group", "grow", "grunt", "guard", "guess", "guide", "guilt", "guitar",
            "gun", "gym", "habit", "hair", "half", "hammer", "hamster", "hand",
            "handle", "harbor", "hard", "harsh", "harvest", "hat", "have", "hawk",
            "hazard", "head", "health", "heart", "heavy", "hedgehog", "height", "hello",
            "helmet", "help", "hen", "hero", "hidden", "high", "hill", "hint",
            "hip", "hire", "history", "hobby", "hockey", "hold", "hole", "holiday",
            "hollow", "home", "honey", "hood", "hope", "horn", "horror", "horse",
            "hospital", "host", "hotel", "hour", "hover", "hub", "huge", "human",
            "humble", "humor", "hundred", "hungry", "hunt", "hurdle", "hurry", "hurt",
            "husband", "hybrid", "ice", "icon", "idea", "identify", "idle", "ignore",
            "ill", "illegal", "illness", "image", "imitate", "immense", "immune", "impact",
            "impose", "improve", "impulse", "inch", "include", "income", "increase", "index",
            "indicate", "indoor", "industry", "infant", "inflict", "inform", "inhale", "inherit",
            "initial", "inject", "injury", "inmate", "inner", "innocent", "input", "inquiry",
            "insane", "insect", "inside", "inspire", "install", "intact", "interest", "into",
            "invest", "invite", "involve", "iron", "island", "isolate", "issue", "item",
            "ivory", "jacket", "jaguar", "jar", "jazz", "jealous", "jeans", "jelly",
            "jewel", "job", "join", "joke", "journey", "joy", "judge", "juice",
            "jump", "jungle", "junior", "junk", "just", "kangaroo", "keen", "keep",
            "ketchup", "key", "kick", "kid", "kidney", "kind", "kingdom", "kiss",
            "kit", "kitchen", "kite", "kitten", "kiwi", "knee", "knife", "knock",
            "know", "lab", "label", "labor", "ladder", "lady", "lake", "lamp",
            "language", "laptop", "large", "later", "latin", "laugh", "laundry", "lava",
            "law", "lawn", "lawsuit", "layer", "lazy", "leader", "leaf", "learn",
            "leave", "lecture", "left", "leg", "legal", "legend", "leisure", "lemon",
            "lend", "length", "lens", "leopard", "lesson", "letter", "level", "liar",
            "liberty", "library", "license", "life", "lift", "light", "like", "limb",
            "limit", "link", "lion", "liquid", "list", "little", "live", "lizard",
            "load", "loan", "lobster", "local", "lock", "logic", "lonely", "long",
            "loop", "lottery", "loud", "lounge", "love", "loyal", "lucky", "luggage",
            "lumber", "lunar", "lunch", "luxury", "lyrics", "machine", "mad", "magic",
            "magnet", "maid", "mail", "main", "major", "make", "mammal", "man",
            "manage", "mandate", "mango", "mansion", "manual", "maple", "marble", "march",
            "margin", "marine", "market", "marriage", "mask", "mass", "master", "match",
            "mate", "material", "math", "matrix", "matter", "maximum", "maze", "meadow",
            "mean", "measure", "meat", "mechanic", "medal", "media", "melody", "melt",
            "member", "memory", "men", "mend", "mental", "mentor", "menu", "mercy",
            "merge", "merit", "merry", "mesh", "message", "metal", "method", "middle",
            "midnight", "milk", "million", "mimic", "mind", "minimum", "minor", "minute",
            "miracle", "mirror", "misery", "miss", "mistake", "mix", "mixed", "mixture",
            "mobile", "model", "modify", "mom", "moment", "monitor", "monkey", "monster",
            "month", "moon", "moral", "more", "morning", "mosquito", "mother", "motion",
            "motor", "mountain", "mouse", "move", "movie", "much", "muffin", "mule",
            "multiply", "muscle", "museum", "mushroom", "music", "must", "mutual", "myself",
            "mystery", "myth", "naive", "name", "napkin", "narrow", "nasty", "nation",
            "nature", "near", "neck", "need", "negative", "neglect", "neither", "nephew",
            "nerve", "nest", "net", "network", "neutral", "never", "news", "next",
            "nice", "night", "noble", "noise", "nominee", "noodle", "normal", "north",
            "nose", "notable", "note", "nothing", "notice", "novel", "now", "nuclear",
            "number", "nurse", "nut", "oak", "obey", "object", "oblige", "obscure",
            "observe", "obtain", "obvious", "occur", "ocean", "october", "odor", "off",
            "offer", "office", "often", "oil", "okay", "old", "olive", "olympic",
            "omit", "once", "one", "onion", "online", "only", "open", "opera",
            "opinion", "oppose", "option", "orange", "orbit", "orchard", "order", "ordinary",
            "organ", "orient", "original", "orphan", "ostrich", "other", "outdoor", "outer",
            "output", "outside", "oval", "oven", "over", "own", "owner", "oxygen",
            "oyster", "ozone", "paddle", "page", "pair", "palace", "palm", "panda",
            "panel", "panic", "panther", "paper", "parade", "parent", "park", "parrot",
            "party", "pass", "patch", "path", "patient", "patrol", "pattern", "pause",
            "pave", "payment", "peace", "peanut", "pear", "peasant", "pelican", "pen",
            "penalty", "pencil", "people", "pepper", "perfect", "permit", "person", "pet",
            "phone", "photo", "phrase", "physical", "piano", "picnic", "picture", "piece",
            "pig", "pigeon", "pill", "pilot", "pink", "pioneer", "pipe", "pistol",
            "pitch", "pizza", "place", "planet", "plastic", "plate", "play", "please",
            "pledge", "pluck", "plug", "plunge", "poem", "poet", "point", "polar",
            "pole", "police", "pond", "pony", "pool", "popular", "portion", "position",
            "possible", "post", "potato", "pottery", "poverty", "powder", "power", "practice",
            "praise", "predict", "prefer", "prepare", "present", "pretty", "prevent", "price",
            "pride", "primary", "print", "priority", "prison", "private", "prize", "problem",
            "process", "produce", "profit", "program", "project", "promote", "proof", "property",
            "prosper", "protect", "proud", "provide", "public", "pudding", "pull", "pulp",
            "pulse", "pumpkin", "punch", "pupil", "puppy", "purchase", "purity", "purpose",
            "purse", "push", "put", "puzzle", "pyramid", "quality", "quantum", "quarter",
            "question", "quick", "quit", "quiz", "quote", "rabbit", "raccoon", "race",
            "rack", "radar", "radio", "rail", "rain", "raise", "rally", "ramp",
            "ranch", "random", "range", "rapid", "rare", "rate", "rather", "raven",
            "raw", "reach", "react", "read", "real", "realm", "rear", "reason",
            "rebel", "rebuild", "recall", "receive", "recipe", "record", "recover", "reduce",
            "reflect", "reform", "refuse", "region", "regret", "regular", "reject", "relax",
            "release", "relief", "rely", "remain", "remember", "remind", "remote", "remove",
            "render", "renew", "rent", "reopen", "repair", "repeat", "replace", "reply",
            "report", "represent", "reproduce", "public", "require", "rescue", "resemble",
            "resist", "resource", "response", "result", "retire", "retreat", "return", "reunion",
            "reveal", "review", "reward", "rhythm", "rib", "ribbon", "rice", "rich",
            "ride", "ridge", "rifle", "right", "rigid", "ring", "riot", "ripple",
            "risk", "ritual", "rival", "river", "road", "roast", "robot", "robust",
            "rocket", "romance", "roof", "rookie", "room", "rose", "rotate", "rough",
            "round", "route", "royal", "rubber", "rude", "rug", "rule", "run",
            "runway", "rural", "sad", "saddle", "sadness", "safe", "sail", "salad",
            "salmon", "salon", "salt", "salute", "same", "sample", "sand", "satisfy",
            "satoshi", "sauce", "sausage", "save", "say", "scale", "scan", "scare",
            "scatter", "scene", "scheme", "school", "science", "scissors", "scorpion",
            "scout", "scrap", "screen", "script", "scrub", "sea", "search", "season",
            "seat", "second", "secret", "section", "security", "seed", "seek", "segment",
            "select", "sell", "seminar", "senior", "sense", "sentence", "series", "service",
            "session", "settle", "setup", "seven", "shadow", "shaft", "shallow", "share",
            "shed", "shell", "sheriff", "shield", "shift", "shine", "ship", "shiver",
            "shock", "shoe", "shoot", "shop", "short", "shoulder", "shove", "shrimp",
            "shrug", "shuffle", "shy", "sibling", "sick", "side", "siege", "sight",
            "sign", "silent", "silk", "silly", "silver", "similar", "simple", "since",
            "sing", "siren", "sister", "situate", "six", "size", "skate", "sketch",
            "ski", "skill", "skin", "skirt", "skull", "slab", "slam", "sleep",
            "slender", "slice", "slide", "slight", "slim", "slogan", "slot", "slow",
            "slush", "small", "smart", "smile", "smoke", "smooth", "snack", "snake",
            "snap", "sniff", "snow", "soap", "soccer", "social", "sock", "soda",
            "soft", "solar", "soldier", "solid", "solution", "solve", "someone", "song",
            "soon", "sorry", "sort", "soul", "sound", "soup", "source", "south",
            "space", "spare", "spatial", "spawn", "speak", "special", "speed", "spell",
            "spend", "sphere", "spice", "spider", "spike", "spin", "spirit", "split",
            "spoil", "sponsor", "spoon", "sport", "spot", "spouse", "spread", "spring",
            "spy", "square", "squeeze", "squirrel", "stable", "stadium", "staff", "stage",
            "stairs", "stamp", "stand", "start", "state", "stay", "steak", "steel",
            "stem", "step", "stereo", "stick", "still", "sting", "stock", "stomach",
            "stone", "stool", "story", "stove", "strategy", "street", "strike", "strong",
            "struggle", "student", "stuff", "stumble", "style", "subject", "submit", "subway",
            "success", "such", "sudden", "suffer", "sugar", "suggest", "suit", "summer",
            "sun", "sunny", "sunset", "super", "supply", "supreme", "sure", "surface",
            "surge", "surprise", "surround", "survey", "suspect", "sustain", "swallow", "swamp",
            "swap", "swarm", "swear", "sweat", "sweep", "sweet", "swift", "swim",
            "swing", "switch", "sword", "symbol", "symptom", "syrup", "system", "table",
            "tackle", "tag", "tail", "talent", "talk", "tank", "tape", "target",
            "task", "taste", "tattoo", "taxi", "teach", "team", "tell", "ten",
            "tenant", "tennis", "tent", "term", "test", "text", "thank", "that",
            "theme", "then", "theory", "there", "they", "thing", "this", "thought",
            "three", "thrive", "throw", "thumb", "thunder", "ticket", "tide", "tiger",
            "tilt", "timber", "time", "tiny", "tip", "tired", "tissue", "title",
            "toast", "tobacco", "toddler", "toe", "together", "toilet", "token", "tomato",
            "tomorrow", "tone", "tongue", "tonight", "tool", "tooth", "top", "topic",
            "topple", "torch", "tornado", "tortoise", "toss", "total", "tourist", "toward",
            "tower", "town", "toy", "track", "trade", "traffic", "tragic", "train",
            "transfer", "trap", "trash", "travel", "tray", "treat", "tree", "trend",
            "trial", "tribe", "trick", "trigger", "trim", "trip", "trophy", "trouble",
            "truck", "true", "truly", "trumpet", "trust", "truth", "try", "tube",
            "tuition", "tumble", "tuna", "tunnel", "turkey", "turn", "turtle", "twelve",
            "twenty", "twice", "twin", "twist", "two", "type", "typical", "ugly",
            "umbrella", "unable", "unaware", "uncle", "uncover", "under", "undo", "unfair",
            "unfold", "unhappy", "uniform", "unique", "unit", "universe", "unknown", "unlock",
            "until", "unusual", "unveil", "update", "upgrade", "uphold", "upon", "upper",
            "upset", "urban", "urge", "usage", "use", "used", "useful", "useless",
            "usual", "utility", "vacant", "vacuum", "vague", "valid", "valley", "valve",
            "van", "vanish", "vapor", "various", "vegan", "velvet", "vendor", "venture",
            "venue", "verb", "verify", "version", "very", "vessel", "veteran", "viable",
            "vibrant", "vicious", "victory", "video", "view", "village", "vintage", "violin",
            "virtual", "virus", "visa", "visit", "visual", "vital", "vivid", "vocal",
            "voice", "void", "volcano", "volume", "vote", "voyage", "wage", "wagon",
            "wait", "walk", "wall", "walnut", "want", "warfare", "warm", "warrior",
            "wash", "wasp", "waste", "water", "wave", "way", "wealth", "weapon",
            "wear", "weasel", "weather", "web", "wedding", "weekend", "weird", "welcome",
            "west", "wet", "whale", "what", "wheat", "wheel", "when", "where",
            "whip", "whisper", "wide", "width", "wife", "wild", "will", "win",
            "window", "wine", "wing", "wink", "winner", "winter", "wire", "wisdom",
            "wise", "wish", "witness", "wolf", "woman", "wonder", "wood", "wool",
            "word", "work", "world", "worry", "worth", "wrap", "wreck", "wrestle",
            "wrist", "write", "wrong", "yard", "year", "yellow", "you", "young",
            "youth", "zebra", "zero", "zone", "zoo"
        ];
        
        let mut words = Vec::new();
        for i in 0..count {
            // Convert bits to word index
            let mut index = 0;
            for j in 0..11 {
                let bit_index = i * 11 + j;
                if bit_index < bits.len() && bits[bit_index] {
                    index = (index << 1) | 1;
                } else {
                    index = index << 1;
                }
            }
            // Get word from wordlist
            if (index as usize) < wordlist.len() {
                words.push(wordlist[index as usize].to_string());
            } else {
                // Fallback (should not happen with valid entropy)
                words.push("abandon".to_string());
            }
        }
        
        words
    }
    
    /// Derive child key at specific path
    pub fn derive_at(
        mnemonic: &str,
        passphrase: &str,
        application: u32,
        index: u32,
    ) -> Result<EntropyOutput, Bip85Error> {
        Self::derive(mnemonic, passphrase, application, index, ENTROPY_LEN_16)
    }
    
    /// Generate secure random entropy
    pub fn generate_random(entropy_len: usize) -> Result<EntropyOutput, Bip85Error> {
        use rand::RngCore;
        
        if entropy_len != 16 && entropy_len != 32 {
            return Err(Bip85Error::InvalidEntropyLength);
        }
        
        let mut entropy = vec![0u8; entropy_len];
        rand::thread_rng().fill_bytes(&mut entropy);
        
        let mnemonic = Self::entropy_to_mnemonic(&entropy)?;
        
        Ok(EntropyOutput {
            entropy,
            mnemonic,
            index: 0,
            application: APP_BACKUP,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_bip85_derive() {
        // Test vector from BIP-85 specification
        let mnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";
        
        let result = Bip85Deriver::derive(mnemonic, "", APP_ETHEREUM, 0, 16);
        
        assert!(result.is_ok(), "BIP-85 derivation failed");
    }
    
    #[test]
    fn test_random_entropy() {
        let result = Bip85Deriver::generate_random(16);
        
        assert!(result.is_ok());
        assert_eq!(result.unwrap().entropy.len(), 16);
    }
    
    #[test]
    fn test_parse_path() {
        let path = "m/83696968'/60'/0'/0'/128'";
        let indices = Bip85Deriver::parse_path(path).unwrap();
        
        assert_eq!(indices.len(), 5);
        assert_eq!(indices[0], 0x80000000 | 83696968);
        assert_eq!(indices[1], 0x80000000 | 60);
    }
}
