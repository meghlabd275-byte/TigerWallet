//! TigerWallet BIP-39 Mnemonic & Key Derivation System
//! 
//! Complete BIP-39 implementation with:
//! - 24-word and 12-word mnemonic generation
//! - BIP-32 HD wallet derivation
//! - BIP-44 multi-chain address derivation
//! - Hardware Security Module (HSM) simulation
//! - Multi-signature support
//! - Social recovery preparation

use std::collections::HashMap;

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum MnemonicError {
    #[error("Invalid mnemonic: {0}")]
    InvalidMnemonic(String),
    
    #[error("Invalid passphrase: {0}")]
    InvalidPassphrase(String),
    
    #[error("Derivation failed: {0}")]
    DerivationFailed(String),
    
    #[error("Invalid path: {0}")]
    InvalidPath(String),
    
    #[error("HSM error: {0}")]
    HSMError(String),
}

pub type Result<T> = std::result::Result<T, MnemonicError>;

// ============================================================================
// BIP-39 Word List (2048 words - English)
// ============================================================================

pub const BIP39_WORDLIST: &[&str; 2048] = &[
    "abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
    "absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
    "acoustic", "acquire", "across", "action", "actor", "actress", "actual", "adapt",
    "add", "addict", "address", "adjust", "admit", "adult", "advance", "advice",
    "aerobic", "affair", "afford", "afraid", "again", "age", "agent", "agree",
    "ahead", "aim", "airport", "airport", "aisle", "alarm", "album", "alcohol",
    "alert", "alien", "all", "alley", "allow", "almost", "alone", "alpha",
    "already", "also", "alter", "always", "amateur", "amazing", "among", "amount",
    "amused", "analyst", "anchor", "ancient", "anger", "angle", "angry", "animal",
    "ankle", "announce", "annual", "another", "answer", "antenna", "antique",
    "anxiety", "any", "apart", "apology", "appear", "apple", "approve", "april",
    "arch", "arctic", "area", "arena", "argue", "arm", "armed", "armor",
    "army", "around", "arrange", "arrest", "arrive", "arrow", "art", "artefact",
    "artist", "artwork", "ask", "aspect", "assault", "asset", "assist", "assume",
    "asthma", "athlete", "atom", "attack", "attend", "attitude", "attract",
    "auction", "audit", "august", "aunt", "author", "auto", "autumn", "average",
    "avocado", "avoid", "awake", "aware", "away", "awesome", "awful", "awkward",
    "axis", "baby", "bachelor", "bacon", "badge", "bag", "balance", "balcony",
    "ball", "bamboo", "banana", "banner", "bar", "barely", "bargain", "barrel",
    "base", "basic", "basket", "battle", "beach", "bean", "beauty", "because",
    "become", "beef", "before", "begin", "behave", "behind", "believe", "below",
    "belt", "bench", "benefit", "best", "betray", "better", "between", "beyond",
    "bicycle", "bid", "bike", "bind", "biology", "bird", "birth", "bitter",
    "black", "blade", "blame", "blanket", "blast", "blaze", "blend", "bless",
    "blind", "blood", "blossom", "blouse", "blue", "blur", "blush", "board",
    "boat", "body", "boil", "bomb", "bone", "bonus", "book", "boost",
    "border", "boring", "borrow", "boss", "bottom", "bounce", "box", "boy",
    "bracket", "brain", "brand", "brass", "brave", "bread", "breeze", "brick",
    "bridge", "brief", "bright", "bring", "brisk", "broccoli", "broken", "bronze",
    "broom", "brother", "brown", "brush", "bubble", "buddy", "budget", "buffalo",
    "build", "bulb", "bulk", "bullet", "bundle", "bunker", "burden", "burger",
    "burst", "bus", "business", "busy", "butter", "buyer", "buzz", "cabbage",
    "cabin", "cable", "cactus", "cage", "cake", "call", "calm", "camera",
    "camp", "can", "canal", "cancel", "candy", "cannon", "canoe", "canvas",
    "canyon", "capable", "capital", "captain", "car", "carbon", "card", "cargo",
    "carpet", "carry", "cart", "case", "cash", "casino", "castle", "casual",
    "cat", "catalog", "catch", "category", "cattle", "caught", "cause", "caution",
    "cave", "ceiling", "celery", "cement", "census", "century", "cereal", "certain",
    "chair", "chalk", "champion", "change", "chaos", "chapter", "charge", "chase",
    "chat", "cheap", "check", "cheese", "chef", "cherry", "chest", "chicken",
    "chief", "child", "china", "chocolate", "choice", "choose", "chronic", "chunk",
    "church", "cigar", "circle", "citizen", "city", "civil", "claim", "clap",
    "clarify", "classic", "clean", "clerk", "clever", "click", "client", "cliff",
    "climb", "clinic", "clip", "clock", "close", "cloth", "cloud", "clown",
    "club", "clump", "cluster", "clutch", "coach", "coast", "coconut", "code",
    "coffee", "coil", "coin", "collect", "color", "column", "combine", "come",
    "comfort", "comic", "common", "company", "concert", "conduct", "confirm", "congress",
    "connect", "consider", "control", "convince", "cook", "cool", "copper", "copy",
    "coral", "core", "corn", "correct", "cost", "cotton", "couch", "country",
    "couple", "course", "cousin", "cover", "coyote", "crack", "cradle", "craft",
    "cram", "crane", "crash", "crater", "crawl", "crazy", "cream", "credit",
    "creek", "crew", "cricket", "crime", "crisp", "critic", "crop", "cross",
    "crouch", "crowd", "crucial", "cruel", "cruise", "crumble", "crunch", "crush",
    "cry", "crystal", "cube", "culture", "cup", "cupboard", "curious", "current",
    "curtain", "curve", "cushion", "custom", "cute", "cycle", "dad", "damage",
    "damp", "dance", "danger", "daring", "dash", "daughter", "dawn", "day",
    "deal", "debate", "debris", "decade", "december", "decide", "decline", "decorate",
    "decrease", "deer", "defense", "define", "defy", "degree", "delay", "deliver",
    "demand", "denial", "dentist", "deny", "depart", "depend", "deposit", "depth",
    "deputy", "derive", "describe", "desert", "design", "desk", "despair", "destroy",
    "detail", "detect", "develop", "device", "devote", "diagram", "dial", "diamond",
    "diary", "dice", "diesel", "diet", "differ", "digital", "dignity", "dilemma",
    "dinner", "dinosaur", "direct", "dirt", "disagree", "discover", "disease", "dish",
    "dismiss", "disorder", "display", "distance", "divert", "divide", "divorce", "dizzy",
    "doctor", "document", "dog", "doll", "dolphin", "domain", "donate", "donkey",
    "donor", "door", "dose", "dot", "double", "dove", "draft", "dragon",
    "drama", "draw", "dream", "dress", "drift", "drill", "drink", "drip",
    "drive", "drop", "drum", "dry", "duck", "dumb", "dune", "during",
    "dust", "dutch", "duty", "dwarf", "dynamic", "eager", "eagle", "early",
    "earn", "earth", "easily", "east", "easy", "echo", "ecology", "economy",
    "edge", "edit", "educate", "effort", "egg", "eight", "eject", "elastic",
    "elbow", "elder", "electric", "elegant", "element", "elephant", "elevator", "elite",
    "else", "embark", "embody", "embrace", "embryo", "emerge", "emotion", "employ",
    "empower", "empty", "enable", "enact", "end", "endless", "endorse", "enemy",
    "energy", "enforce", "engage", "engine", "enhance", "enjoy", "enlist", "enough",
    "enrich", "enroll", "ensure", "enter", "entire", "entry", "envelope", "episode",
    "equal", "equip", "era", "erase", "erode", "erosion", "error", "erupt",
    "escape", "essay", "essence", "estate", "eternal", "ethics", "eviction", "evidence",
    "evil", "evoke", "evolve", "exact", "exceed", "except", "excess", "exchange",
    "excite", "exclude", "excuse", "execute", "exercise", "exhaust", "exhibit", "exile",
    "exist", "exit", "exotic", "expand", "expect", "expire", "explain", "expose",
    "express", "extend", "extra", "exterior", "external", "extra", "extreme", "eye",
    "eyebrow", "fabric", "face", "faculty", "fade", "faint", "faith", "fall",
    "false", "fame", "family", "famous", "fan", "fancy", "fantasy", "farm",
    "fashion", "fat", "fatal", "father", "fatigue", "fault", "favorite", "feature",
    "february", "federal", "fee", "feed", "feel", "female", "fence", "festival",
    "fetch", "fever", "few", "fiber", "fiction", "field", "figure", "file",
    "film", "filter", "final", "finance", "find", "fine", "finger", "finish",
    "fire", "firm", "first", "fiscal", "fish", "fist", "fit", "fitness",
    "fix", "flag", "flame", "flash", "flat", "flavor", "flea", "flight",
    "flip", "float", "flock", "flood", "floor", "flower", "fluid", "flush",
    "fly", "foam", "focus", "fog", "foil", "fold", "folk", "follow",
    "food", "foot", "force", "forest", "forget", "fork", "fortune", "forum",
    "forward", "fossil", "found", "fox", "fragile", "frame", "frequent", "fresh",
    "friend", "fringe", "frog", "front", "frost", "frown", "frozen", "fruit",
    "fuel", "fun", "funny", "furnace", "fury", "future", "gadget", "gain",
    "galaxy", "gallery", "game", "gap", "garage", "garbage", "garden", "garlic",
    "gas", "gasp", "gate", "gather", "gauge", "gaze", "general", "genius",
    "genre", "gentle", "genuine", "gesture", "ghost", "giant", "gift", "giggle",
    "ginger", "giraffe", "girl", "give", "glad", "glance", "glare", "glass",
    "glide", "glimpse", "globe", "gloom", "glory", "glove", "glow", "glue",
    "goat", "goddess", "gold", "good", "goose", "gorilla", "gospel", "gossip",
    "govern", "gown", "grab", "grace", "grain", "grant", "grape", "grass",
    "gravity", "great", "green", "grid", "grief", "grit", "grocery", "group",
    "grow", "grunt", "guard", "guess", "guide", "guilt", "guitar", "gun",
    "gym", "habit", "hair", "half", "hammer", "hamster", "hand", "handle",
    "harbor", "hard", "harsh", "harvest", "hat", "have", "hawk", "hazard",
    "head", "health", "heart", "heavy", "hedgehog", "height", "hello", "helmet",
    "help", "hen", "hero", "hidden", "high", "hill", "hint", "hip",
    "hire", "history", "hobby", "hockey", "hold", "hole", "holiday", "hollow",
    "home", "honest", "honey", "honor", "hope", "horn", "horror", "horse",
    "hospital", "host", "hotel", "hour", "hover", "hub", "huge", "human",
    "humble", "humor", "hundred", "hungry", "hunt", "hurdle", "hurry", "hurt",
    "husband", "hybrid", "ice", "icon", "idea", "identify", "idle", "ignore",
    "ill", "illegal", "illness", "image", "imitate", "immense", "immune", "impact",
    "impose", "improve", "impulse", "inch", "include", "income", "increase", "index",
    "indicate", "indoor", "industry", "infant", "inflict", "inform", "inhale", "inherit",
    "initial", "inject", "injury", "inmate", "inner", "innocent", "input", "inquiry",
    "insane", "insect", "inside", "inspire", "install", "intact", "interest", "into",
    "invest", "invite", "involve", "iris", "iron", "island", "isolate", "issue",
    "item", "ivory", "jacket", "jaguar", "jar", "jazz", "jealous", "jeans",
    "jelly", "jewel", "job", "jog", "join", "joint", "joke", "journal",
    "journey", "joy", "judge", "juice", "jump", "jungle", "junior", "junk",
    "just", "kangaroo", "keen", "keep", "ketchup", "key", "kick", "kid",
    "kidney", "kind", "kingdom", "kiss", "kit", "kitchen", "kite", "kitten",
    "kiwi", "knee", "knife", "knock", "know", "lab", "label", "labor",
    "ladder", "lady", "lake", "lamp", "language", "laptop", "large", "later",
    "latin", "laugh", "laundry", "lava", "law", "lawn", "lawsuit", "layer",
    "lazy", "leader", "leaf", "learn", "leave", "lecture", "left", "leg",
    "legal", "legend", "lemon", "lend", "length", "lens", "leopard", "lesson",
    "letter", "level", "liar", "liberty", "library", "license", "life", "lift",
    "light", "like", "limb", "limit", "link", "lion", "liquid", "list",
    "little", "live", "lizard", "load", "loan", "lobster", "local", "lock",
    "logic", "lonely", "long", "loop", "lottery", "loud", "lounge", "love",
    "loyal", "lucky", "luggage", "lumber", "lunar", "lunch", "luxury", "lyrics",
    "machine", "mad", "magic", "magnet", "maid", "mail", "main", "major",
    "make", "mammal", "man", "manage", "mandate", "mango", "mansion", "manual",
    "maple", "marble", "march", "margin", "marine", "market", "marriage", "mask",
    "mass", "master", "match", "material", "math", "matrix", "matter", "maximum",
    "maze", "meadow", "mean", "measure", "meat", "mechanic", "medal", "media",
    "melody", "melt", "member", "memory", "men", "mend", "mental", "mentor",
    "menu", "mercy", "merge", "merit", "merry", "mesh", "message", "metal",
    "method", "middle", "midnight", "milk", "million", "mimic", "mind", "minimum",
    "minor", "minute", "miracle", "mirror", "misery", "miss", "mistake", "mix",
    "mixed", "mixture", "mobile", "model", "modify", "mom", "moment", "monitor",
    "monkey", "monster", "month", "moon", "moral", "more", "morning", "mosquito",
    "mother", "motion", "motor", "mountain", "mouse", "move", "movie", "much",
    "muffin", "mule", "multiply", "muscle", "museum", "mushroom", "music", "must",
    "mutual", "myself", "mystery", "myth", "naive", "name", "napkin", "narrow",
    "nasty", "nation", "nature", "near", "neck", "need", "negative", "neglect",
    "neither", "nephew", "nerve", "nest", "net", "network", "neutral", "never",
    "news", "next", "nice", "night", "noble", "noise", "nominee", "noodle",
    "normal", "north", "nose", "notable", "note", "nothing", "notice", "novel",
    "now", "nuclear", "number", "nurse", "nut", "oak", "obey", "object",
    "oblige", "obscure", "observe", "obtain", "obvious", "occur", "ocean", "october",
    "odor", "off", "offer", "office", "often", "oil", "okay", "old",
    "olive", "olympic", "omit", "once", "one", "onion", "online", "only",
    "open", "opera", "opinion", "oppose", "option", "orange", "orbit", "orchard",
    "order", "ordinary", "organ", "orient", "original", "orphan", "ostrich", "other",
    "outdoor", "outer", "output", "oval", "oven", "over", "own", "owner",
    "oxygen", "oyster", "ozone", "pact", "paddle", "page", "pair", "palace",
    "palm", "panda", "panel", "panic", "panther", "paper", "parade", "paramount",
    "parent", "park", "parrot", "party", "pass", "patch", "path", "patient",
    "patrol", "pattern", "pause", "pave", "payment", "peace", "peanut", "pear",
    "peasant", "penny", "people", "pepper", "perfect", "permit", "person", "pet",
    "phone", "photo", "phrase", "physical", "piano", "picnic", "picture", "piece",
    "pig", "pigeon", "pill", "pilot", "pink", "pioneer", "pipe", "pistol",
    "pitch", "pizza", "place", "planet", "plastic", "plate", "play", "please",
    "pledge", "plenty", "plot", "plough", "pluck", "plug", "plunge", "poem",
    "poet", "point", "polar", "pole", "police", "pond", "pony", "pool",
    "popular", "portion", "position", "possible", "post", "potato", "pottery", "poverty",
    "powder", "power", "practice", "praise", "predict", "prefer", "prepare", "present",
    "pretty", "prevent", "price", "pride", "primary", "print", "priority", "prison",
    "private", "prize", "problem", "process", "produce", "profit", "program", "project",
    "promote", "proof", "property", "prosper", "protect", "proud", "provide", "public",
    "pudding", "pull", "pulp", "pulse", "pumpkin", "punch", "pupil", "puppy",
    "purchase", "purity", "purpose", "purse", "push", "put", "puzzle", "pyramid",
    "quality", "quantum", "quarter", "question", "quick", "quiet", "quilt", "quota",
    "quote", "rabbit", "raccoon", "race", "rack", "radar", "radio", "rail", "rain",
    "raise", "rally", "ramp", "ranch", "random", "range", "rapid", "rare", "rate",
    "rather", "raven", "raw", "reach", "react", "read", "reader", "real",
    "reality", "realize", "realm", "rear", "reason", "rebel", "rebuild", "recall",
    "receipt", "receive", "recipe", "record", "recover", "recruit", "red", "reduce",
    "reflect", "reform", "refuse", "region", "regret", "regular", "reject", "relate",
    "relax", "release", "relief", "rely", "remain", "remember", "remind", "remote",
    "remove", "render", "renew", "rent", "reopen", "repair", "repeat", "replace",
    "reply", "report", "represent", "reproduce", "public", "require", "rescue", "resemble",
    "resist", "resource", "response", "result", "retire", "retreat", "return", "reunion",
    "reveal", "review", "reward", "rhythm", "rib", "ribbon", "rice", "rich",
    "ride", "ridge", "rifle", "right", "rigid", "ring", "riot", "ripple",
    "risk", "ritual", "rival", "river", "road", "roast", "robot", "robust",
    "rocket", "romance", "roof", "rookie", "room", "rose", "rotate", "rough",
    "round", "route", "royal", "rubber", "rude", "rug", "rule", "run",
    "runway", "rural", "sad", "saddle", "sadness", "safe", "sail", "salad",
    "salmon", "salon", "salt", "salute", "same", "sample", "sand", "satisfy",
    "satoshi", "sauce", "sausage", "save", "say", "scale", "scan", "scare",
    "scatter", "scene", "scheme", "school", "science", "scissors", "scorpion", "scout",
    "scrap", "screen", "script", "scrub", "sea", "search", "season", "seat",
    "second", "secret", "section", "security", "seed", "seek", "segment", "seize", "select",
    "self", "sell", "seminar", "senior", "sense", "sentence", "series", "service",
    "session", "settle", "setup", "seven", "shadow", "shaft", "shallow", "share", "shed",
    "shell", "sheriff", "shield", "shift", "shine", "ship", "shiver", "shock",
    "shoe", "shoot", "shop", "short", "shoulder", "shove", "shrimp", "shrug",
    "shuffle", "shun", "shut", "sibling", "sick", "side", "siege", "sight",
    "sign", "silent", "silicon", "silk", "silly", "silver", "similar", "simple",
    "since", "sing", "siren", "sister", "situate", "six", "sixteen", "size",
    "skate", "sketch", "ski", "skill", "skin", "skirt", "skull", "slab",
    "slam", "sleep", "slice", "slide", "slight", "slim", "slogan", "slot",
    "slow", "slush", "small", "smart", "smell", "smile", "smoke", "smooth",
    "snack", "snake", "snap", "sniff", "snow", "so", "soap", "soccer",
    "social", "sock", "soda", "soft", "solar", "soldier", "sole", "some",
    "son", "song", "soon", "sorry", "sort", "soul", "soup", "source", "south",
    "space", "spare", "spark", "speak", "special", "speed", "spell", "spend",
    "sphere", "spice", "spider", "spike", "spin", "spirit", "split", "spoil",
    "sponsor", "spoon", "sport", "spot", "spray", "spread", "spring", "spy",
    "square", "squeeze", "squirrel", "stable", "stadium", "staff", "stage", "stairs",
    "stamp", "stand", "start", "state", "stay", "steak", "steel", "stem",
    "step", "stereo", "stick", "still", "sting", "stock", "stomach", "stone",
    "stool", "story", "stove", "strategy", "street", "strike", "strong", "struggle",
    "student", "stuff", "stumble", "style", "subject", "submit", "subway", "success",
    "such", "sudden", "suffer", "sugar", "suggest", "suit", "summer", "sun",
    "sunny", "sunset", "super", "supply", "supreme", "sure", "surface", "surge",
    "surprise", "surround", "survey", "suspect", "sustain", "swallow", "swamp", "swap",
    "swarm", "swear", "sweat", "sweep", "sweet", "swift", "swim", "swing",
    "switch", "sword", "symbol", "symptom", "syrup", "system", "table", "tackle",
    "tag", "tail", "talent", "talk", "tank", "tape", "target", "task",
    "taste", "tattoo", "taxi", "teach", "team", "tell", "ten", "tenant",
    "tennis", "tense", "tent", "term", "test", "text", "thank", "that", "theme",
    "then", "theory", "there", "they", "thing", "this", "thought", "three",
    "thrive", "throw", "thumb", "thunder", "ticket", "tide", "tiger", "tilt",
    "timber", "time", "tiny", "tip", "tired", "tissue", "title", "toast",
    "tobacco", "toddler", "toe", "together", "toilet", "token", "tomato", "tomorrow",
    "tone", "tongue", "tonight", "tool", "tooth", "top", "topic", "topple",
    "torch", "tornado", "tortoise", "toss", "total", "tourist", "toward", "tower",
    "town", "toy", "track", "trade", "traffic", "tragic", "train", "transfer",
    "transform", "transit", "translate", "trap", "trash", "travel", "tray", "treat",
    "tree", "trend", "trial", "tribe", "trick", "trigger", "trim", "trip",
    "trophy", "trouble", "truck", "true", "truly", "trumpet", "trust", "truth",
    "try", "tube", "tuition", "tumble", "tuna", "tunnel", "turkey", "turn",
    "turtle", "twelve", "twenty", "twice", "twin", "twist", "two", "type",
    "typical", "ugly", "umbrella", "unable", "unaware", "uncle", "uncover", "under",
    "undo", "unfair", "unfold", "unhappy", "uniform", "unique", "unit", "universe",
    "unknown", "unlock", "until", "unusual", "unveil", "update", "upgrade", "uphold",
    "upon", "upper", "upset", "urban", "urge", "usage", "use", "used",
    "useful", "useless", "usual", "utility", "vacant", "vacuum", "vague", "valid",
    "valley", "valve", "vanilla", "vanish", "various", "vegan", "velvet", "vendor",
    "venture", "venue", "verb", "verify", "version", "very", "vessel", "veteran", "viable",
    "vibrant", "vicious", "victory", "video", "view", "village", "vintage", "violin",
    "virtual", "virus", "visa", "visit", "visual", "vital", "vivid", "vocal", "voice",
    "void", "volcano", "volume", "vote", "voyage", "wage", "wagon", "wait", "walk",
    "wall", "walnut", "want", "warfare", "warm", "warrior", "wash", "wasp", "waste",
    "watch", "water", "wave", "way", "wealth", "weapon", "wear", "weasel", "weather",
    "web", "wedding", "weekend", "weird", "welcome", "west", "wet", "whale", "what",
    "wheat", "wheel", "when", "where", "whip", "whisper", "whistle", "white", "who",
    "whole", "why", "wicked", "wide", "widow", "width", "wife", "wild", "will",
    "win", "window", "wine", "wing", "wink", "winner", "winter", "wire", "wisdom",
    "wise", "wish", "witness", "wolf", "woman", "wonder", "wood", "wool", "word",
    "work", "world", "worry", "worth", "wrap", "wreck", "wrestle", "wrist", "write",
    "wrong", "yard", "year", "yell", "yellow", "you", "young", "youth", "zebra",
    "zero", "zone", "zoo",
];

// ============================================================================
// Mnemonic Generation
// ============================================================================

/// Compute entropy from mnemonic (for verification)
fn mnemonic_to_entropy(mnemonic: &[&str]) -> Result<Vec<u8>> {
    if mnemonic.len() != 12 && mnemonic.len() != 24 {
        return Err(MnemonicError::InvalidMnemonic(
            format!("Mnemonic must be 12 or 24 words, got {}", mnemonic.len())
        ));
    }
    
    // Join words and hash to get entropy
    let mut entropy = Vec::new();
    for word in mnemonic {
        let idx = BIP39_WORDLIST.iter().position(|w| *w == *word);
        match idx {
            Some(i) => {
                // Each word represents 11 bits
                entropy.extend_from_slice(&(i as u16).to_be_bytes());
            },
            None => {
                return Err(MnemonicError::InvalidMnemonic(
                    format!("Invalid word: {}", word)
                ));
            }
        }
    }
    
    Ok(entropy)
}

/// Generate random mnemonic (12 or 24 words)
pub fn generate_mnemonic(word_count: usize) -> Result<Vec<String>> {
    if word_count != 12 && word_count != 24 {
        return Err(MnemonicError::InvalidMnemonic(
            format!("Word count must be 12 or 24, got {}", word_count)
        ));
    }
    
    // Generate random entropy
    let entropy_bits = word_count * 8 / 3; // 128 or 256 bits
    let entropy = super::generate_secure_random(entropy_bits / 8)?;
    
    // Convert entropy to words using checksum
    let entropy_hash = super::sha256(&entropy);
    let checksum_bits = word_count / 3; // 4 or 8 bits
    
    let mut words = Vec::new();
    for i in 0..word_count {
        // Get 11-bit index from entropy + checksum
        let bit_pos = i * 11;
        let byte_pos = bit_pos / 8;
        let bit_offset = bit_pos % 8;
        
        let mut value: u16 = if byte_pos < entropy.len() {
            (entropy[byte_pos] as u16) << 8
        } else {
            (entropy_hash[0] as u16) << 8
        };
        
        if byte_pos + 1 < entropy.len() {
            value |= entropy[byte_pos + 1] as u16;
        }
        
        if byte_pos + 1 >= entropy.len() && byte_pos < entropy.len() {
            // Use checksum for last word(s)
            value = (entropy_hash[0] as u16) << 8 | (entropy_hash[1] as u16);
        }
        
        value = value >> (16 - 11 - bit_offset) & 0x7FF;
        
        if (value as usize) < BIP39_WORDLIST.len() {
            words.push(BIP39_WORDLIST[value as usize].to_string());
        }
    }
    
    Ok(words)
}

// ============================================================================
// BIP-39 Seed Derivation
// ============================================================================

/// Convert mnemonic to BIP-39 seed
pub fn mnemonic_to_seed(mnemonic: &[String], passphrase: &str) -> Result<[u8; 64]> {
    use pbkdf2::pbkdf2_hmac_array;
    use sha2::Sha512;
    
    // Validate mnemonic
    for word in mnemonic {
        if !BIP39_WORDLIST.contains(word) {
            return Err(MnemonicError::InvalidMnemonic(
                format!("Unknown word: {}", word)
            ));
        }
    }
    
    let mnemonic_str = mnemonic.join(" ");
    let salt = format!("mnemonic{}", passphrase);
    
    // PBKDF2-SHA512 with 2048 iterations
    let seed = pbkdf2_hmac_array::<Sha512, 64>(
        mnemonic_str.as_bytes(),
        salt.as_bytes(),
        2048,
    );
    
    Ok(seed)
}

// ============================================================================
// BIP-32 HD Wallet
// ============================================================================

/// HD Wallet key structure
#[derive(Clone, Debug)]
pub struct HDKey {
    pub key: [u8; 32],       // Private key (or public key for public derivation)
    pub chain_code: [u8; 32], // Chain code for child derivation
    pub public_key: Option<[u8; 33]>, // Compressed public key
}

/// Create root HD key from seed
pub fn hd_key_from_seed(seed: &[u8; 64]) -> Result<HDKey> {
    use hmac::{Hmac, Mac};
    use sha2::Sha512;
    
    type HmacSha512 = Hmac<Sha512>;
    
    // I = HMAC-SHA512(Key = "Bitcoin seed", Data = seed)
    let mut mac = HmacSha512::new_from_slice(b"Bitcoin seed")
        .map_err(|e| MnemonicError::DerivationFailed(e.to_string()))?;
    mac.update(seed);
    
    let result = mac.finalize().into_bytes();
    
    let mut key = [0u8; 32];
    let mut chain_code = [0u8; 32];
    
    key.copy_from_slice(&result[..32]);
    chain_code.copy_from_slice(&result[32..]);
    
    // Generate public key
    let public_key = super::private_to_public(&key);
    let mut compressed = [0u8; 33];
    compressed[0] = 0x02; // Even Y coordinate
    compressed[1..].copy_from_slice(&public_key[..32]);
    
    Ok(HDKey {
        key,
        chain_code,
        public_key: Some(compressed),
    })
}

/// Derive child key (BIP-32)
pub fn derive_child_key(parent: &HDKey, index: u32) -> Result<HDKey> {
    let is_hardened = index >= 0x80000000;
    
    let mut data = Vec::new();
    
    if is_hardened {
        // Hardened derivation: 0x00 || parent.key || index
        data.push(0x00);
        data.extend_from_slice(&parent.key);
    } else {
        // Normal derivation: parent.public_key || index
        if let Some(ref pk) = parent.public_key {
            data.extend_from_slice(pk);
        }
    }
    
    data.extend_from_slice(&index.to_be_bytes());
    
    // HMAC-SHA512
    use hmac::{Hmac, Mac};
    use sha2::Sha512;
    
    type HmacSha512 = Hmac<Sha512>;
    
    let mut mac = HmacSha512::new_from_slice(&parent.chain_code)
        .map_err(|e| MnemonicError::DerivationFailed(e.to_string()))?;
    mac.update(&data);
    
    let result = mac.finalize().into_bytes();
    
    let mut child_key = [0u8; 32];
    let mut child_chain_code = [0u8; 32];
    
    child_key.copy_from_slice(&result[..32]);
    child_chain_code.copy_from_slice(&result[32..]);
    
    // Add to parent key
    for i in 0..32 {
        child_key[i] ^= parent.key[i];
    }
    
    // Generate public key
    let public_key = super::private_to_public(&child_key);
    let mut compressed = [0u8; 33];
    compressed[0] = 0x02;
    compressed[1..].copy_from_slice(&public_key[..32]);
    
    Ok(HDKey {
        key: child_key,
        chain_code: child_chain_code,
        public_key: Some(compressed),
    })
}

// ============================================================================
// BIP-44 Path Derivation
// ============================================================================

/// BIP-44 purpose constant
pub const BIP44_PURPOSE: u32 = 0x8000002C; // 44' (hardened)
/// BIP-44 coin types
pub const COIN_ETH: u32 = 0x8000035C;    // 60' (Ethereum)
pub const COIN_BTC: u32 = 0x80000000;    // 0' (Bitcoin)
pub const COIN_SOL: u32 = 0x8000012C;    // 501' (Solana)
pub const COIN_COSMOS: u32 = 0x8000014B;   // 118' (Cosmos)
pub const COIN_POLKADOT: u32 = 0x80000154; // 354' (Polkadot)

/// BIP-44 path structure
#[derive(Clone, Debug)]
pub struct BIP44Path {
    pub purpose: u32,
    pub coin_type: u32,
    pub account: u32,
    pub change: u32,
    pub address_index: u32,
}

impl BIP44Path {
    /// Parse path string like "m/44'/60'/0'/0/0"
    pub fn from_string(path: &str) -> Result<Self> {
        let parts: Vec<&str> = path.trim_start_matches("m/").split('/').collect();
        
        if parts.len() != 5 {
            return Err(MnemonicError::InvalidPath(format!("Invalid path: {}", path)));
        }
        
        let purpose = parse_path_component(parts[0])?;
        let coin_type = parse_path_component(parts[1])?;
        let account = parse_path_component(parts[2])?;
        let change = parse_path_component(parts[3])?;
        let address_index = parse_path_component(parts[4])?;
        
        Ok(BIP44Path {
            purpose,
            coin_type,
            account,
            change,
            address_index,
        })
    }
    
    /// Derive key from root using this path
    pub fn derive(&self, root: &HDKey) -> Result<HDKey> {
        let key = derive_child_key(root, self.purpose)?;
        let key = derive_child_key(&key, self.coin_type)?;
        let key = derive_child_key(&key, self.account)?;
        let key = derive_child_key(&key, self.change)?;
        let key = derive_child_key(&key, self.address_index)?;
        Ok(key)
    }
    
    /// Get Ethereum path: m/44'/60'/0'/0/0
    pub fn ethereum(account: u32, address_index: u32) -> Self {
        BIP44Path {
            purpose: BIP44_PURPOSE,
            coin_type: COIN_ETH,
            account,
            change: 0,
            address_index,
        }
    }
    
    /// Get Bitcoin path: m/44'/0'/0'/0/0
    pub fn bitcoin(account: u32, address_index: u32) -> Self {
        BIP44Path {
            purpose: BIP44_PURPOSE,
            coin_type: COIN_BTC,
            account,
            change: 0,
            address_index,
        }
    }
    
    /// Get Solana path: m/44'/501'/0'/0'
    pub fn solana(account: u32, address_index: u32) -> Self {
        BIP44Path {
            purpose: BIP44_PURPOSE,
            coin_type: COIN_SOL,
            account,
            change: 0,
            address_index,
        }
    }
}

fn parse_path_component(s: &str) -> Result<u32> {
    let s = s.trim_end_matches('\'');
    s.parse::<u32>()
        .map_err(|e| MnemonicError::InvalidPath(format!("Invalid component: {}", e)))
}

// ============================================================================
// Hardware Security Module (HSM) Simulation
// ============================================================================

/// HSM Key Store - simulates hardware security module
pub struct HSMKeyStore {
    keys: HashMap<String, HSMKeyEntry>,
}

#[derive(Clone)]
struct HSMKeyEntry {
    key: Vec<u8>,
    created_at: u64,
    usage_count: u32,
}

impl HSMKeyStore {
    pub fn new() -> Self {
        HSMKeyStore {
            keys: HashMap::new(),
        }
    }
    
    /// Generate key in HSM (simulated)
    pub fn generate_key(&mut self, key_id: &str) -> Result<()> {
        let key = super::generate_key()
            .map_err(|e| MnemonicError::HSMError(e.to_string()))?;
        
        self.keys.insert(key_id.to_string(), HSMKeyEntry {
            key: key.to_vec(),
            created_at: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs() as u64,
            usage_count: 0,
        });
        
        Ok(())
    }
    
    /// Sign with HSM key (simulated)
    pub fn sign(&mut self, key_id: &str, message: &[u8]) -> Result<Vec<u8>> {
        let entry = self.keys.get_mut(key_id)
            .ok_or_else(|| MnemonicError::HSMError("Key not found".to_string()))?;
        
        entry.usage_count += 1;
        
        // Sign with the key
        let key_array: [u8; 32] = entry.key.as_slice().try_into()
            .map_err(|_| MnemonicError::HSMError("Invalid key length".to_string()))?;
        
        // Use simple HMAC for simulation (real HSM would use ECDSA)
        Ok(super::hmac_sha256(&key_array, message).to_vec())
    }
    
    /// Delete key from HSM
    pub fn delete_key(&mut self, key_id: &str) -> Result<()> {
        self.keys.remove(key_id)
            .ok_or_else(|| MnemonicError::HSMError("Key not found".to_string()))?;
        Ok(())
    }
}

impl Default for HSMKeyStore {
    fn default() -> Self {
        Self::new()
    }
}

// ============================================================================
// Multi-Signature Wallet Support
// ============================================================================

/// Multi-signature key set
#[derive(Clone)]
pub struct MultisigKeySet {
    pub threshold: u8,
    pub pubkeys: Vec<Vec<u8>>,
}

impl MultisigKeySet {
    /// Create new multi-sig with threshold
    pub fn new(threshold: u8, pubkeys: Vec<Vec<u8>>) -> Result<Self> {
        if threshold == 0 || threshold > pubkeys.len() as u8 {
            return Err(MnemonicError::InvalidKey(
                "Invalid threshold".to_string()
            ));
        }
        
        Ok(MultisigKeySet { threshold, pubkeys })
    }
    
    /// Combine signatures (simplified - real implementation would be ECDSA)
    pub fn combine_signatures(&self, signatures: &[Vec<u8>]) -> Result<Vec<u8>> {
        if signatures.len() < self.threshold as usize {
            return Err(MnemonicError::DerivationFailed(
                format!("Need {} signatures, got {}", self.threshold, signatures.len())
            ));
        }
        
        // Simplified: concatenate signatures
        let mut combined = Vec::new();
        for sig in signatures.iter().take(self.threshold as usize) {
            combined.extend_from_slice(sig);
            combined.push(b',');
        }
        combined.pop(); // Remove trailing comma
        
        Ok(combined)
    }
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_generate_mnemonic_12() {
        let mnemonic = generate_mnemonic(12).unwrap();
        assert_eq!(mnemonic.len(), 12);
        println!("12-word mnemonic: {:?}", mnemonic);
    }
    
    #[test]
    fn test_generate_mnemonic_24() {
        let mnemonic = generate_mnemonic(24).unwrap();
        assert_eq!(mnemonic.len(), 24);
        println!("24-word mnemonic: {:?}", mnemonic);
    }
    
    #[test]
    fn test_mnemonic_to_seed() {
        let mnemonic = vec![
            "abandon".to_string(), "abandon".to_string(), "abandon".to_string(),
            "abandon".to_string(), "abandon".to_string(), "abandon".to_string(),
            "abandon".to_string(), "abandon".to_string(), "abandon".to_string(),
            "abandon".to_string(), "abandon".to_string(), "abandon".to_string(),
        ];
        
        let seed = mnemonic_to_seed(&mnemonic, "").unwrap();
        assert_eq!(seed.len(), 64);
    }
    
    #[test]
    fn test_hd_key_derivation() {
        let seed = super::generate_key().unwrap();
        let root = hd_key_from_seed(&seed).unwrap();
        
        // Derive first child
        let child = derive_child_key(&root, 0).unwrap();
        
        assert!(!child.key.iter().all(|&b| b == 0));
    }
    
    #[test]
    fn test_bip44_path() {
        let path = BIP44Path::ethereum(0, 0);
        assert_eq!(path.purpose, BIP44_PURPOSE);
        assert_eq!(path.coin_type, COIN_ETH);
        
        // Parse from string
        let path2 = BIP44Path::from_string("m/44'/60'/0'/0/0").unwrap();
        assert_eq!(path.purpose, path2.purpose);
    }
    
    #[test]
    fn test_hsm() {
        let mut hsm = HSMKeyStore::new();
        
        hsm.generate_key("test-key").unwrap();
        let sig = hsm.sign("test-key", b"message").unwrap();
        
        assert_eq!(sig.len(), 32);
        
        hsm.delete_key("test-key").unwrap();
    }
    
    #[test]
    fn test_multisig() {
        let pubkeys = vec![
            vec![1u8; 33],
            vec![2u8; 33],
            vec![3u8; 33],
        ];
        
        let ms = MultisigKeySet::new(2, pubkeys).unwrap();
        
        let sigs = vec![vec![1u8; 32], vec![2u8; 32]];
        let combined = ms.combine_signatures(&sigs).unwrap();
        
        assert!(!combined.is_empty());
    }
}