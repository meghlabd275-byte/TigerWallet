//! TigerWallet Mnemonic Engine - Rust Implementation
//! BIP-39 Mnemonic Phrase Generation and Validation
//! Supports 12, 15, 18, 21, and 24 word phrases

use std::collections::HashMap;

// ============================================================================
// BIP-39 Word List (English - 2048 words)
// ============================================================================

pub const BIP39_WORDLIST: &[&str; 2048] = &[
    "abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
    "absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
    "acoustic", "acquire", "across", "action", "actor", "actress", "actual", "adapt",
    "add", "addict", "address", "adjust", "admit", "adult", "advance", "advice",
    "aerobic", "affair", "afford", "afraid", "again", "age", "agent", "agree",
    "ahead", "aim", "airport", "aisle", "alarm", "album", "alcohol", "alert",
    "alien", "all", "alley", "allow", "almost", "alone", "alpha", "already",
    "also", "alter", "always", "amateur", "amazing", "among", "amount", "amused",
    "analyst", "anchor", "ancient", "anger", "angle", "angry", "animal", "ankle",
    "announce", "annual", "another", "answer", "antenna", "antique", "anxiety", "any",
    "apart", "apology", "appear", "apple", "approve", "april", "arch", "arctic",
    "area", "arena", "argue", "arm", "armed", "armor", "army", "around", "arrange",
    "arrest", "arrive", "arrow", "art", "artefact", "artist", "artwork", "ask",
    "aspect", "assault", "asset", "assist", "assume", "asthma", "athlete", "atom",
    "attack", "attend", "attitude", "attract", "auction", "audit", "august", "aunt",
    "author", "auto", "autumn", "average", "avocado", "avoid", "awake", "aware",
    "away", "awesome", "awful", "awkward", "axis", "baby", "bachelor", "bacon",
    "badge", "bag", "balance", "balcony", "ball", "bamboo", "banana", "banner",
    "bar", "barely", "bargain", "barrel", "base", "basic", "basket", "battle",
    "beach", "bean", "beauty", "because", "become", "beef", "before", "begin",
    "behave", "behind", "believe", "below", "belt", "bench", "benefit", "best",
    "betray", "better", "between", "beyond", "bicycle", "bid", "bike", "bind",
    "biology", "bird", "birth", "bitter", "black", "blade", "blame", "blanket",
    "blast", "blaze", "blend", "bless", "blind", "blood", "blossom", "blouse",
    "blue", "blur", "blush", "board", "boat", "body", "boil", "bomb", "bone",
    "bonus", "book", "boost", "border", "boring", "borrow", "boss", "bottom",
    "bounce", "box", "boy", "bracket", "brain", "brand", "brass", "brave",
    "bread", "breeze", "brick", "bridge", "brief", "bright", "bring", "brisk",
    "broccoli", "broken", "bronze", "broom", "brother", "brown", "brush", "bubble",
    "buddy", "budget", "buffalo", "build", "bulb", "bulk", "bullet", "bundle",
    "bunker", "burden", "burger", "burst", "bus", "business", "busy", "butter",
    "buyer", "buzz", "cabbage", "cabin", "cable", "cactus", "cage", "cake",
    "call", "calm", "camera", "camp", "can", "canal", "cancel", "candy", "cannon",
    "canoe", "canvas", "canyon", "capable", "capital", "captain", "car", "carbon",
    "card", "cargo", "carpet", "carry", "cart", "case", "cash", "casino",
    "castle", "casual", "cat", "catalog", "catch", "category", "cattle", "caught",
    "cause", "caution", "cave", "ceiling", "celery", "cement", "census", "century",
    "cereal", "certain", "chair", "chalk", "champion", "change", "chaos",
    "chapter", "charge", "chase", "chat", "cheap", "check", "cheese", "chef",
    "cherry", "chest", "chicken", "chief", "child", "china", "chocolate",
    "choice", "choose", "chronic", "chunk", "church", "cigar", "circle",
    "citizen", "city", "civil", "claim", "clap", "clarify", "classic", "clean",
    "clerk", "clever", "click", "client", "cliff", "climb", "clinic", "clip",
    "clock", "close", "cloth", "cloud", "clown", "club", "clump", "cluster",
    "clutch", "coach", "coast", "coconut", "code", "coffee", "coil", "coin",
    "collect", "color", "column", "combine", "come", "comfort", "comic", "common",
    "company", "concert", "conduct", "confirm", "congress", "connect", "consider",
    "control", "convince", "cook", "cool", "copper", "copy", "coral", "core",
    "corn", "correct", "cost", "cottage", "cotton", "couch", "country", "couple",
    "course", "cousin", "cover", "coyote", "crack", "cradle", "craft", "cram",
    "crane", "crash", "crater", "crawl", "crazy", "cream", "credit", "creek",
    "crew", "cricket", "crime", "crisp", "critic", "crop", "cross", "crouch",
    "crowd", "crucial", "cruel", "cruise", "crumble", "crunch", "crush", "cry",
    "crystal", "cube", "culture", "cup", "cupboard", "curious", "current",
    "curtain", "curve", "cushion", "custom", "cute", "cycle", "dad", "damage",
    "damp", "dance", "danger", "daring", "dash", "daughter", "dawn", "day",
    "deal", "debate", "debris", "decade", "december", "decide", "decline",
    "decorate", "decrease", "deer", "defense", "define", "defy", "degree",
    "delay", "deliver", "demand", "denial", "dentist", "deny", "depart",
    "depend", "deposit", "depth", "deputy", "derive", "describe", "desert",
    "design", "desk", "despair", "destroy", "detail", "detect", "develop",
    "device", "devote", "diagram", "dial", "diamond", "diary", "dice", "diesel",
    "diet", "differ", "digital", "dignity", "dilemma", "dinner", "dinosaur",
    "direct", "dirt", "disagree", "discover", "disease", "dish", "dismiss",
    "disorder", "display", "distance", "divert", "divide", "divorce", "dizzy",
    "doctor", "document", "dog", "doll", "dolphin", "domain", "donate", "donkey",
    "donor", "door", "dose", "dot", "double", "dove", "draft", "dragon",
    "drama", "draw", "dream", "dress", "drift", "drill", "drink", "drip",
    "drive", "drop", "drum", "dry", "duck", "dumb", "dune", "during",
    "dust", "dutch", "duty", "dwarf", "dynamic", "eager", "eagle", "early",
    "earn", "earth", "easily", "east", "easy", "echo", "ecology", "economy",
    "edge", "edit", "educate", "effort", "egg", "eight", "eject", "elastic",
    "elbow", "elder", "electric", "elegant", "element", "elephant", "elevator",
    "elite", "else", "embark", "embody", "embrace", "embryo", "emerge",
    "emotion", "employ", "empower", "empty", "enable", "enact", "end", "endless",
    "endorse", "enemy", "energy", "enforce", "engage", "engine", "enhance",
    "enjoy", "enlist", "enough", "enrich", "enroll", "ensure", "enter",
    "entire", "entry", "envelope", "episode", "equal", "equip", "era", "erase",
    "erode", "erosion", "error", "erupt", "escape", "essay", "essence",
    "estate", "eternal", "ethics", "eviction", "evidence", "evil", "evoke",
    "evolve", "exact", "exceed", "except", "excess", "exchange", "excite",
    "exclude", "excuse", "execute", "exercise", "exhaust", "exhibit", "exile",
    "exist", "exit", "exotic", "expand", "expect", "expire", "explain",
    "expose", "express", "extend", "extra", "exterior", "external", "extreme",
    "eye", "eyebrow", "fabric", "face", "faculty", "fade", "faint", "faith",
    "fall", "false", "fame", "family", "famous", "fan", "fancy", "fantasy",
    "farm", "fashion", "fat", "fatal", "father", "fatigue", "fault",
    "favorite", "feature", "february", "federal", "fee", "feed", "feel",
    "female", "fence", "festival", "fetch", "fever", "few", "fiber",
    "fiction", "field", "figure", "file", "film", "filter", "final", "finance",
    "find", "fine", "finger", "finish", "fire", "firm", "first", "fiscal",
    "fish", "fist", "fit", "fitness", "fix", "flag", "flame", "flash",
    "flat", "flavor", "flea", "flight", "flip", "float", "flock", "flood",
    "floor", "flower", "fluid", "flush", "fly", "foam", "focus", "fog",
    "foil", "fold", "folk", "follow", "food", "foot", "force", "forest",
    "forget", "fork", "fortune", "forum", "forward", "fossil", "found",
    "fox", "fragile", "frame", "frequent", "fresh", "friend", "fringe",
    "frog", "front", "frost", "frown", "frozen", "fruit", "fuel", "fun",
    "funny", "furnace", "fury", "future", "gadget", "gain", "galaxy",
    "gallery", "game", "gap", "garage", "garbage", "garden", "garlic", "gas",
    "gasp", "gate", "gather", "gauge", "gaze", "general", "genius", "genre",
    "gentle", "genuine", "gesture", "ghost", "giant", "gift", "giggle",
    "ginger", "giraffe", "girl", "give", "glad", "glance", "glare", "glass",
    "glide", "glimpse", "globe", "gloom", "glory", "glove", "glow", "glue",
    "goat", "goddess", "gold", "good", "goose", "gorilla", "gospel",
    "gossip", "govern", "gown", "grab", "grace", "grain", "grant", "grape",
    "grass", "gravity", "great", "green", "grid", "grief", "grit", "grocery",
    "group", "grow", "grunt", "guard", "guess", "guide", "guilt", "guitar",
    "gun", "gym", "habit", "hair", "half", "hammer", "hamster", "hand",
    "handle", "harbor", "hard", "harsh", "harvest", "hat", "have", "hawk",
    "hazard", "head", "health", "heart", "heavy", "hedgehog", "height",
    "hello", "helmet", "help", "hen", "hero", "hidden", "high", "hill",
    "hint", "hip", "hire", "history", "hobby", "hockey", "hold", "hole",
    "holiday", "hollow", "home", "honest", "honey", "honor", "hope",
    "horn", "horror", "horse", "hospital", "host", "hotel", "hour", "hover",
    "hub", "huge", "human", "humble", "humor", "hundred", "hungry",
    "hunt", "hurdle", "hurry", "hurt", "husband", "hybrid", "ice", "icon",
    "idea", "identify", "idle", "ignore", "ill", "illegal", "illness",
    "image", "imitate", "immense", "immune", "impact", "impose", "improve",
    "impulse", "inch", "include", "income", "increase", "index", "indicate",
    "indoor", "industry", "infant", "inflict", "inform", "inhale", "inherit",
    "initial", "inject", "injury", "inmate", "inner", "innocent", "input",
    "inquiry", "insane", "insect", "inside", "inspire", "install", "intact",
    "interest", "into", "invest", "invite", "involve", "iris", "iron",
    "island", "isolate", "issue", "item", "ivory", "jacket", "jaguar",
    "jar", "jazz", "jealous", "jeans", "jelly", "jewel", "job", "jog",
    "join", "joint", "joke", "journal", "journey", "joy", "judge", "juice",
    "jump", "jungle", "junior", "junk", "just", "kangaroo", "keen",
    "keep", "ketchup", "key", "kick", "kid", "kidney", "kind", "kingdom",
    "kiss", "kit", "kitchen", "kite", "kitten", "kiwi", "knee", "knife",
    "knock", "know", "lab", "label", "labor", "ladder", "lady", "lake",
    "lamp", "language", "laptop", "large", "later", "latin", "laugh",
    "laundry", "lava", "law", "lawn", "lawsuit", "layer", "lazy", "leader",
    "leaf", "learn", "leave", "lecture", "left", "leg", "legal", "legend",
    "lemon", "lend", "length", "lens", "leopard", "lesson", "letter",
    "level", "liar", "liberty", "library", "license", "life", "lift",
    "light", "like", "limb", "limit", "link", "lion", "liquid", "list",
    "little", "live", "lizard", "load", "loan", "lobster", "local",
    "lock", "logic", "lonely", "long", "loop", "lottery", "loud", "lounge",
    "love", "loyal", "lucky", "luggage", "lumber", "lunar", "lunch",
    "luxury", "lyrics", "machine", "mad", "magic", "magnet", "maid",
    "mail", "main", "major", "make", "mammal", "man", "manage", "mandate",
    "mango", "mansion", "manual", "maple", "marble", "march", "margin",
    "marine", "market", "marriage", "mask", "mass", "master", "match",
    "material", "math", "matrix", "matter", "maximum", "maze", "meadow",
    "mean", "measure", "meat", "mechanic", "medal", "media", "melody",
    "melt", "member", "memory", "men", "mend", "mental", "mentor", "menu",
    "mercy", "merge", "merit", "merry", "mesh", "message", "metal",
    "method", "middle", "midnight", "milk", "million", "mimic", "mind",
    "minimum", "minor", "minute", "miracle", "mirror", "misery", "miss",
    "mistake", "mix", "mixed", "mixture", "mobile", "model", "modify",
    "mom", "moment", "monitor", "monkey", "monster", "month", "moon",
    "moral", "more", "morning", "mosquito", "mother", "motion", "motor",
    "mountain", "mouse", "move", "movie", "much", "muffin", "mule",
    "multiply", "muscle", "museum", "mushroom", "music", "must", "mutual",
    "myself", "mystery", "myth", "naive", "name", "napkin", "narrow",
    "nasty", "nation", "nature", "near", "neck", "need", "negative",
    "neglect", "neither", "nephew", "nerve", "nest", "net", "network",
    "neutral", "never", "news", "next", "nice", "night", "noble", "noise",
    "nominee", "noodle", "normal", "north", "nose", "notable", "note",
    "nothing", "notice", "novel", "now", "nuclear", "number", "nurse",
    "nut", "oak", "obey", "object", "oblige", "obscure", "observe",
    "obtain", "obvious", "occur", "ocean", "october", "odor", "off",
    "offer", "office", "often", "oil", "okay", "old", "olive", "olympic",
    "omit", "once", "one", "onion", "online", "only", "open", "opera",
    "opinion", "oppose", "option", "orange", "orbit", "orchard", "order",
    "ordinary", "organ", "orient", "original", "orphan", "ostrich", "other",
    "outdoor", "outer", "output", "oval", "oven", "over", "own",
    "owner", "oxygen", "oyster", "ozone", "pact", "paddle", "page",
    "pair", "palace", "palm", "panda", "panel", "panic", "panther",
    "paper", "parade", "paramount", "parent", "park", "parrot", "party",
    "pass", "patch", "path", "patient", "patrol", "pattern", "pause",
    "pave", "payment", "peace", "peanut", "pear", "peasant", "penny",
    "people", "pepper", "perfect", "permit", "person", "pet", "phone",
    "photo", "phrase", "physical", "piano", "picnic", "picture", "piece",
    "pig", "pigeon", "pill", "pilot", "pink", "pioneer", "pipe",
    "pistol", "pitch", "pizza", "place", "planet", "plastic", "plate",
    "play", "please", "pledge", "plenty", "plot", "plough", "pluck",
    "plug", "plunge", "poem", "poet", "point", "polar", "pole", "police",
    "pond", "pony", "pool", "popular", "portion", "position", "possible",
    "post", "potato", "pottery", "poverty", "powder", "power", "practice",
    "praise", "predict", "prefer", "prepare", "present", "pretty", "prevent",
    "price", "pride", "primary", "print", "priority", "prison", "private",
    "prize", "problem", "process", "produce", "profit", "program", "project",
    "promote", "proof", "property", "prosper", "protect", "proud", "provide",
    "public", "pudding", "pull", "pulp", "pulse", "pumpkin", "punch",
    "pupil", "puppy", "purchase", "purity", "purpose", "purse", "push",
    "put", "puzzle", "pyramid", "quality", "quantum", "quarter", "question",
    "quick", "quiet", "quilt", "quota", "quote", "rabbit", "raccoon",
    "race", "rack", "radar", "radio", "rail", "rain", "raise", "rally",
    "ramp", "ranch", "random", "range", "rapid", "rare", "rate", "rather",
    "raven", "raw", "reach", "react", "read", "reader", "real", "reality",
    "realize", "realm", "rear", "reason", "rebel", "rebuild", "recall",
    "receipt", "receive", "recipe", "record", "recover", "recruit", "red",
    "reduce", "reflect", "reform", "refuse", "region", "regret", "regular",
    "reject", "relate", "relax", "release", "relief", "rely", "remain",
    "remember", "remind", "remote", "remove", "render", "renew", "rent",
    "reopen", "repair", "repeat", "replace", "reply", "report", "represent",
    "reproduce", "public", "require", "rescue", "resemble", "resist",
    "resource", "response", "result", "retire", "retreat", "return",
    "reunion", "reveal", "review", "reward", "rhythm", "rib", "ribbon",
    "rice", "rich", "ride", "ridge", "rifle", "right", "rigid", "ring",
    "riot", "ripple", "risk", "ritual", "rival", "river", "road",
    "roast", "robot", "robust", "rocket", "romance", "roof", "rookie",
    "room", "rose", "rotate", "rough", "round", "route", "royal", "rubber",
    "rude", "rug", "rule", "run", "runway", "rural", "sad", "saddle",
    "sadness", "safe", "sail", "salad", "salmon", "salon", "salt",
    "salute", "same", "sample", "sand", "satisfy", "satoshi", "sauce",
    "sausage", "save", "say", "scale", "scan", "scare", "scatter",
    "scene", "scheme", "school", "science", "scissors", "scorpion", "scout",
    "scrap", "screen", "script", "scrub", "sea", "search", "season",
    "seat", "second", "secret", "section", "security", "seed", "seek",
    "segment", "seize", "select", "self", "sell", "seminar", "senior",
    "sense", "sentence", "series", "service", "session", "settle", "setup",
    "seven", "shadow", "shaft", "shallow", "share", "shed", "shell",
    "sheriff", "shield", "shift", "shine", "ship", "shiver", "shock",
    "shoe", "shoot", "shop", "short", "shoulder", "shove", "shrimp",
    "shrug", "shuffle", "shun", "shut", "sibling", "sick", "side",
    "siege", "sight", "sign", "silent", "silicon", "silk", "silly",
    "silver", "similar", "simple", "since", "sing", "siren", "sister",
    "situate", "six", "sixteen", "size", "skate", "sketch", "ski",
    "skill", "skin", "skirt", "skull", "slab", "slam", "sleep",
    "slice", "slide", "slight", "slim", "slogan", "slot", "slow",
    "slush", "small", "smart", "smell", "smile", "smoke", "smooth",
    "snack", "snake", "snap", "sniff", "snow", "so", "soap", "soccer",
    "social", "sock", "soda", "soft", "solar", "soldier", "sole",
    "some", "son", "song", "soon", "sorry", "sort", "soul", "soup",
    "source", "south", "space", "spare", "spark", "speak", "special",
    "speed", "spell", "spend", "sphere", "spice", "spider", "spike",
    "spin", "spirit", "split", "spoil", "sponsor", "spoon", "sport",
    "spot", "spray", "spread", "spring", "spy", "square", "squeeze",
    "squirrel", "stable", "stadium", "staff", "stage", "stairs", "stamp",
    "stand", "start", "state", "stay", "steak", "steel", "stem", "step",
    "stereo", "stick", "still", "sting", "stock", "stomach", "stone",
    "stool", "story", "stove", "strategy", "street", "strike", "strong",
    "struggle", "student", "stuff", "stumble", "style", "subject",
    "submit", "subway", "success", "such", "sudden", "suffer", "sugar",
    "suggest", "suit", "summer", "sun", "sunny", "sunset", "super",
    "supply", "supreme", "sure", "surface", "surge", "surprise",
    "surround", "survey", "suspect", "sustain", "swallow", "swamp", "swap",
    "swarm", "swear", "sweat", "sweep", "sweet", "swift", "swim",
    "swing", "switch", "sword", "symbol", "symptom", "syrup", "system",
    "table", "tackle", "tag", "tail", "talent", "talk", "tank",
    "tape", "target", "task", "taste", "tattoo", "taxi", "teach",
    "team", "tell", "ten", "tenant", "tennis", "tense", "tent",
    "term", "test", "text", "thank", "that", "theme", "then", "theory",
    "there", "they", "thing", "this", "thought", "three", "thrive",
    "throw", "thumb", "thunder", "ticket", "tide", "tiger", "tilt",
    "timber", "time", "tiny", "tip", "tired", "tissue", "title",
    "toast", "tobacco", "toddler", "toe", "together", "toilet", "token",
    "tomato", "tomorrow", "tone", "tongue", "tonight", "tool", "tooth",
    "top", "topic", "topple", "torch", "tornado", "tortoise", "toss",
    "total", "tourist", "toward", "tower", "town", "toy", "track",
    "trade", "traffic", "tragic", "train", "transfer", "transform",
    "transit", "translate", "trap", "trash", "travel", "tray", "treat",
    "tree", "trend", "trial", "tribe", "trick", "trigger", "trim",
    "trip", "trophy", "trouble", "truck", "true", "truly", "trumpet",
    "trust", "truth", "try", "tube", "tuition", "tumble", "tuna",
    "tunnel", "turkey", "turn", "turtle", "twelve", "twenty", "twice",
    "twin", "twist", "two", "type", "typical", "ugly", "umbrella",
    "unable", "unaware", "uncle", "uncover", "under", "undo", "unfair",
    "unfold", "unhappy", "uniform", "unique", "unit", "universe", "unknown",
    "unlock", "until", "unusual", "unveil", "update", "upgrade", "uphold",
    "upon", "upper", "upset", "urban", "urge", "usage", "use", "used",
    "useful", "useless", "usual", "utility", "vacant", "vacuum",
    "vague", "valid", "valley", "valve", "vanilla", "vanish", "various",
    "vegan", "velvet", "vendor", "venture", "venue", "verb", "verify",
    "version", "very", "vessel", "veteran", "viable", "vibrant", "vicious",
    "victory", "video", "view", "village", "vintage", "violin", "virtual",
    "virus", "visa", "visit", "visual", "vital", "vivid", "vocal",
    "voice", "void", "volcano", "volume", "vote", "voyage", "wage",
    "wagon", "wait", "walk", "wall", "walnut", "want", "warfare",
    "warm", "warrior", "wash", "wasp", "waste", "watch", "water",
    "wave", "way", "wealth", "weapon", "wear", "weasel", "weather",
    "web", "wedding", "weekend", "weird", "welcome", "west", "wet",
    "whale", "what", "wheat", "wheel", "when", "where", "whip",
    "whisper", "whistle", "white", "who", "whole", "why", "wicked",
    "wide", "widow", "width", "wife", "wild", "will", "win",
    "window", "wine", "wing", "wink", "winner", "winter", "wire",
    "wisdom", "wise", "wish", "witness", "wolf", "woman", "wonder",
    "wood", "wool", "word", "work", "world", "worry", "worth",
    "wrap", "wreck", "wrestle", "wrist", "write", "wrong", "yard",
    "year", "yell", "yellow", "you", "young", "youth", "zebra", "zero",
    "zone", "zoo",
];

// ============================================================================
// Mnemonic Engine
// ============================================================================

#[derive(Debug, Clone)]
pub struct MnemonicEngine {
    wordlist: Vec<String>,
}

impl MnemonicEngine {
    /// Create new mnemonic engine
    pub fn new() -> Self {
        MnemonicEngine {
            wordlist: BIP39_WORDLIST.iter().map(|s| s.to_string()).collect(),
        }
    }

    /// Generate mnemonic with specified word count
    pub fn generate(&self, word_count: usize) -> Result<Vec<String>, MnemonicError> {
        if ![12, 15, 18, 21, 24].contains(&word_count) {
            return Err(MnemonicError::InvalidWordCount(
                "Word count must be 12, 15, 18, 21, or 24".to_string()
            ));
        }

        let entropy_bits = word_count * 11 - (word_count / 3);
        let entropy_bytes = entropy_bits / 8;
        
        // Generate random entropy
        let entropy = generate_secure_random(entropy_bytes)?;
        
        // Calculate checksum
        let checksum = sha256_hash(&entropy);
        let checksum_bits = word_count / 3;
        
        // Build mnemonic
        let mut bits = Vec::new();
        for byte in &entropy {
            for i in (0..8).rev() {
                bits.push((byte >> i) & 1);
            }
        }
        
        // Add checksum bits
        for i in 0..checksum_bits {
            let byte_idx = i / 8;
            let bit_idx = 7 - (i % 8);
            bits.push((checksum[byte_idx] >> bit_idx) & 1);
        }
        
        // Convert to words
        let mut words = Vec::new();
        for chunk in bits.chunks(11) {
            let mut value = 0u16;
            for (i, &bit) in chunk.iter().enumerate() {
                if bit == 1 {
                    value |= 1 << (10 - i);
                }
            }
            words.push(self.wordlist[value as usize].clone());
        }
        
        Ok(words)
    }

    /// Generate 12-word mnemonic (128-bit entropy)
    pub fn generate_12(&self) -> Result<Vec<String>, MnemonicError> {
        self.generate(12)
    }

    /// Generate 24-word mnemonic (256-bit entropy)
    pub fn generate_24(&self) -> Result<Vec<String>, MnemonicError> {
        self.generate(24)
    }

    /// Validate mnemonic phrase
    pub fn validate(&self, words: &[String]) -> Result<bool, MnemonicError> {
        if ![12, 15, 18, 21, 24].contains(&words.len()) {
            return Err(MnemonicError::InvalidWordCount(
                "Word count must be 12, 15, 18, 21, or 24".to_string()
            ));
        }

        // Check all words are in wordlist
        let wordset: std::collections::HashSet<_> = self.wordlist.iter().collect();
        for word in words {
            if !wordset.contains(word) {
                return Err(MnemonicError::InvalidWord(format!("Unknown word: {}", word)));
            }
        }

        // Verify checksum
        let entropy_bits = words.len() * 11 - (words.len() / 3);
        let entropy_bytes = entropy_bits / 8;
        
        // Convert words back to entropy
        let mut bits = Vec::new();
        for word in words {
            let idx = self.wordlist.iter().position(|w| w == word).unwrap();
            for i in (0..11).rev() {
                bits.push((idx >> i) & 1);
            }
        }
        
        let mut entropy = vec![0u8; entropy_bytes];
        for (i, chunk) in bits.chunks(8).take(entropy_bytes).enumerate() {
            for (j, &bit) in chunk.iter().enumerate() {
                if bit == 1 {
                    entropy[i] |= 1 << (7 - j);
                }
            }
        }
        
        let checksum = sha256_hash(&entropy);
        let checksum_bits = words.len() / 3;
        
        // Verify checksum bits
        for i in 0..checksum_bits {
            let byte_idx = entropy_bytes + i / 8;
            let bit_idx = 7 - (i % 8);
            let expected = (checksum[0] >> bit_idx) & 1;
            let actual = bits.get(entropy_bits + i).copied().unwrap_or(0);
            if expected != actual {
                return Ok(false);
            }
        }
        
        Ok(true)
    }

    /// Convert mnemonic to seed
    pub fn to_seed(&self, words: &[String], passphrase: &str) -> Result<[u8; 64], MnemonicError> {
        // Validate words
        self.validate(words)?;
        
        let mnemonic = words.join(" ");
        let salt = format!("mnemonic{}", passphrase);
        
        // PBKDF2-SHA512 with 2048 iterations
        let seed = pbkdf2_sha512(mnemonic.as_bytes(), salt.as_bytes(), 2048)?;
        
        Ok(seed)
    }

    /// Find word index
    pub fn find_word(&self, prefix: &str) -> Option<String> {
        let prefix_lower = prefix.to_lowercase();
        self.wordlist.iter()
            .find(|w| w.starts_with(&prefix_lower))
            .cloned()
    }
}

// ============================================================================
// Error Types
// ============================================================================

#[derive(Debug, thiserror::Error)]
pub enum MnemonicError {
    #[error("Invalid word count: {0}")]
    InvalidWordCount(String),
    
    #[error("Invalid word: {0}")]
    InvalidWord(String),
    
    #[error("Random generation failed: {0}")]
    RandomError(String),
    
    #[error("Seed derivation failed: {0}")]
    SeedError(String),
}

// ============================================================================
// Helper Functions
// ============================================================================

fn generate_secure_random(length: usize) -> Result<Vec<u8>, MnemonicError> {
    use rand::rngs::OsRng;
    use rand::RngCore;
    
    let mut bytes = vec![0u8; length];
    OsRng.fill_bytes(&mut bytes);
    
    if bytes.iter().all(|&b| b == 0) {
        return Err(MnemonicError::RandomError("CSPRNG failure".to_string()));
    }
    
    Ok(bytes)
}

fn sha256_hash(data: &[u8]) -> Vec<u8> {
    use sha2::{Sha256, Digest};
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

fn pbkdf2_sha512(password: &[u8], salt: &[u8], iterations: u32) -> Result<[u8; 64], MnemonicError> {
    use pbkdf2::pbkdf2_hmac_array;
    use sha2::Sha512;
    
    let result = pbkdf2_hmac_array::<Sha512, 64>(password, salt, iterations);
    Ok(result)
}

// ============================================================================
// Tests
// ============================================================================

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_12() {
        let engine = MnemonicEngine::new();
        let words = engine.generate_12().unwrap();
        
        assert_eq!(words.len(), 12);
        assert!(engine.validate(&words).unwrap());
    }

    #[test]
    fn test_generate_24() {
        let engine = MnemonicEngine::new();
        let words = engine.generate_24().unwrap();
        
        assert_eq!(words.len(), 24);
        assert!(engine.validate(&words).unwrap());
    }

    #[test]
    fn test_to_seed() {
        let engine = MnemonicEngine::new();
        let words = vec!["abandon".to_string(); 12];
        
        let seed = engine.to_seed(&words, "").unwrap();
        
        assert_eq!(seed.len(), 64);
    }

    #[test]
    fn test_validate() {
        let engine = MnemonicEngine::new();
        
        // Valid 12-word mnemonic
        let words = vec![
            "abandon".to_string(), "abandon".to_string(), "abandon".to_string(),
            "abandon".to_string(), "abandon".to_string(), "abandon".to_string(),
            "abandon".to_string(), "abandon".to_string(), "abandon".to_string(),
            "abandon".to_string(), "abandon".to_string(), "abandon".to_string(),
        ];
        
        assert!(engine.validate(&words).is_ok());
    }
}