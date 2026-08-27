/**
 * Wallet Crypto Module — real BIP-39 mnemonic validation, BIP-32 HD key
 * derivation, and EVM address derivation for the Tauri desktop app.
 *
 * No fake/stub/random addresses: every EVM address returned by this module is
 * deterministically derived from the user's 24-word BIP-39 seed (the same
 * seed that controls every EVM + non-EVM chain simultaneously) via the
 * canonical BIP-44 m/44'/60'/0'/0/<account> path, using HMAC-SHA512 CKD and
 * Keccak-256 over the uncompressed secp256k1 public key — exactly matching
 * the Go wallet_api / MasterWallet derivation (verified against the canonical
 * BIP-39 "abandon...about" vectors).
 */

use hmac::{Hmac, Mac};
use pbkdf2::pbkdf2_hmac;
use sha2::{Digest, Sha256, Sha512};
use sha3::Keccak256;

/// BIP-39 English wordlist (2048 words) — fixed, canonical order. The index of
/// each word in this list defines the 11-bit groups of the mnemonic entropy.
const BIP39_ENGLISH: &[&str] = &[
    "abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract", "absurd",
    "abuse", "access", "accident", "account", "accuse", "achieve", "acid", "acoustic", "acquire",
    "across", "act", "action", "actor", "actress", "actual", "adapt", "add", "addict", "address",
    "adjust", "admit", "adult", "advance", "advice", "aerobic", "affair", "afford", "afraid",
    "again", "age", "agent", "agree", "ahead", "aim", "air", "airport", "aisle", "alarm", "album",
    "alcohol", "alert", "alien", "all", "alley", "allow", "almost", "alone", "alpha", "already",
    "also", "alter", "always", "amateur", "amazing", "among", "amount", "amused", "analyst",
    "anchor", "ancient", "anger", "angle", "angry", "animal", "ankle", "announce", "annual",
    "another", "answer", "antenna", "antique", "anxiety", "any", "apart", "apology", "appear",
    "apple", "approve", "april", "arch", "arctic", "area", "arena", "argue", "arm", "armed",
    "armor", "army", "around", "arrange", "arrest", "arrive", "arrow", "art", "artefact", "artist",
    "artwork", "ask", "aspect", "assault", "asset", "assist", "assume", "asthma", "athlete",
    "atom", "attack", "attend", "attitude", "attract", "auction", "audit", "august", "aunt",
    "author", "auto", "autumn", "average", "avocado", "avoid", "awake", "aware", "away", "awesome",
    "awful", "awkward", "axis", "baby", "bachelor", "bacon", "badge", "bag", "balance", "balcony",
    "ball", "bamboo", "banana", "banner", "bar", "barely", "bargain", "barrel", "base", "basic",
    "basket", "battle", "beach", "bean", "beauty", "because", "become", "beef", "before", "begin",
    "behave", "behind", "believe", "below", "belt", "bench", "benefit", "best", "betray", "better",
    "between", "beyond", "bicycle", "bid", "bike", "bind", "biology", "bird", "birth", "bitter",
    "black", "blade", "blame", "blanket", "blast", "bleak", "bless", "blind", "blood", "blossom",
    "blouse", "blue", "blur", "blush", "board", "boat", "body", "boil", "bomb", "bone", "bonus",
    "book", "boost", "border", "boring", "borrow", "boss", "bottom", "bounce", "box", "boy",
    "bracket", "brain", "brand", "brass", "brave", "bread", "breeze", "brick", "bridge", "brief",
    "bright", "bring", "brisk", "broccoli", "broken", "bronze", "broom", "brother", "brown",
    "brush", "bubble", "buddy", "budget", "buffalo", "build", "bulb", "bulk", "bullet", "bundle",
    "bunker", "burden", "burger", "burst", "bus", "business", "busy", "butter", "buyer", "buzz",
    "cabbage", "cabin", "cable", "cactus", "cage", "cake", "call", "calm", "camera", "camp", "can",
    "canal", "cancel", "candy", "cannon", "canoe", "canvas", "canyon", "capable", "capital",
    "captain", "car", "carbon", "card", "cargo", "carpet", "carry", "cart", "case", "cash",
    "casino", "castle", "casual", "cat", "catalog", "catch", "category", "cattle", "caught",
    "cause", "caution", "cave", "ceiling", "celery", "cement", "census", "century", "cereal",
    "certain", "chair", "chalk", "champion", "change", "chaos", "chapter", "charge", "chase",
    "chat", "cheap", "check", "cheese", "chef", "cherry", "chest", "chicken", "chief", "child",
    "chimney", "choice", "choose", "chronic", "chuckle", "chunk", "churn", "cigar", "cinnamon",
    "circle", "citizen", "city", "civil", "claim", "clap", "clarify", "claw", "clay", "clean",
    "clerk", "clever", "click", "client", "cliff", "climb", "clinic", "clip", "clock", "clog",
    "close", "cloth", "cloud", "clown", "club", "clump", "cluster", "clutch", "coach", "coast",
    "coconut", "code", "coffee", "coil", "coin", "collect", "color", "column", "combine", "come",
    "comfort", "comic", "common", "company", "concert", "conduct", "confirm", "congress", "connect",
    "consider", "control", "convince", "cook", "cool", "copper", "copy", "coral", "core", "corn",
    "correct", "cost", "cotton", "couch", "country", "couple", "course", "cousin", "cover", "coyote",
    "crack", "cradle", "craft", "cram", "crane", "crash", "crater", "crawl", "crazy", "cream",
    "credit", "creek", "crew", "cricket", "crime", "crisp", "critic", "crop", "cross", "crouch",
    "crowd", "crucial", "cruel", "cruise", "crumble", "crunch", "crush", "cry", "crystal", "cube",
    "culture", "cup", "cupboard", "curious", "current", "curtain", "curve", "cushion", "custom",
    "cute", "cycle", "dad", "damage", "damp", "dance", "danger", "daring", "dash", "daughter",
    "dawn", "day", "deal", "debate", "debris", "decade", "december", "decide", "decline", "decorate",
    "decrease", "deer", "defense", "define", "defy", "degree", "delay", "deliver", "demand",
    "demise", "denial", "dentist", "deny", "depart", "depend", "deposit", "depth", "deputy",
    "derive", "describe", "desert", "design", "desk", "despair", "destroy", "detail", "detect",
    "develop", "device", "devote", "diagram", "dial", "diamond", "diary", "dice", "diesel", "diet",
    "differ", "digital", "dignity", "dilemma", "dinner", "dinosaur", "direct", "dirt", "disagree",
    "discover", "disease", "dish", "dismiss", "disorder", "display", "distance", "divert", "divide",
    "divorce", "dizzy", "doctor", "document", "dog", "doll", "dolphin", "domain", "donate", "donkey",
    "donor", "door", "dose", "double", "dove", "draft", "dragon", "drama", "drastic", "draw",
    "dream", "dress", "drift", "drill", "drink", "drip", "drive", "drop", "drum", "dry", "duck",
    "dumb", "dune", "during", "dust", "dutch", "duty", "dwarf", "dynamic", "eager", "eagle",
    "early", "earn", "earth", "easily", "east", "easy", "echo", "ecology", "economy", "edge",
    "edit", "educate", "effort", "egg", "eight", "either", "elbow", "elder", "electric", "elegant",
    "element", "elephant", "elevator", "elite", "else", "embark", "embody", "embrace", "emerge",
    "emotion", "employ", "empower", "empty", "enable", "enact", "end", "endless", "endorse",
    "enemy", "energy", "enforce", "engage", "engine", "enhance", "enjoy", "enlist", "enough",
    "enrich", "enroll", "ensure", "enter", "entire", "entry", "envelope", "episode", "equal",
    "equip", "era", "erase", "erode", "erosion", "error", "erupt", "escape", "essay", "essence",
    "estate", "eternal", "ethics", "evidence", "evil", "evoke", "evolve", "exact", "example",
    "excess", "exchange", "excite", "exclude", "excuse", "execute", "exercise", "exhaust",
    "exhibit", "exile", "exist", "exit", "exotic", "expand", "expect", "expire", "explain",
    "expose", "express", "extend", "extra", "eye", "eyebrow", "fabric", "face", "faculty", "fade",
    "faint", "faith", "fall", "false", "fame", "family", "famous", "fan", "fancy", "fantasy",
    "farm", "fashion", "fat", "fatal", "father", "fatigue", "fault", "favorite", "feature",
    "february", "federal", "fee", "feed", "feel", "female", "fence", "festival", "fetch", "fever",
    "few", "fiber", "fiction", "field", "figure", "file", "film", "filter", "final", "find", "fine",
    "finger", "finish", "fire", "firm", "first", "fiscal", "fish", "fit", "fitness", "fix",
    "flag", "flame", "flash", "flat", "flavor", "flee", "flight", "flip", "float", "flock",
    "floor", "flower", "fluid", "flush", "fly", "foam", "focus", "fog", "foil", "fold", "follow",
    "food", "foot", "force", "forest", "forget", "fork", "fortune", "forum", "forward", "fossil",
    "foster", "found", "fox", "fragile", "frame", "frequent", "fresh", "friend", "fringe", "frog",
    "front", "frost", "frown", "frozen", "fruit", "fuel", "fun", "funny", "furnace", "fury",
    "future", "gadget", "gain", "galaxy", "gallery", "game", "gap", "garage", "garbage", "garden",
    "garlic", "garment", "gas", "gasp", "gate", "gather", "gauge", "gaze", "general", "genius",
    "genre", "gentle", "genuine", "gesture", "ghost", "giant", "gift", "giggle", "ginger",
    "giraffe", "girl", "give", "glad", "glance", "glare", "glass", "glide", "glimpse", "globe",
    "gloom", "glory", "glove", "glow", "glue", "goat", "goddess", "gold", "good", "goose",
    "gorilla", "gospel", "gossip", "govern", "gown", "grab", "grace", "grain", "grant", "grape",
    "grass", "gravity", "great", "green", "grid", "grief", "grit", "grocery", "group", "grow",
    "grunt", "guard", "guess", "guide", "guilt", "guitar", "gun", "gym", "habit", "hair", "half",
    "hammer", "hamster", "hand", "happy", "harbor", "hard", "harsh", "harvest", "hat", "have",
    "hawk", "hazard", "head", "health", "heart", "heavy", "hedgehog", "height", "hello", "helmet",
    "help", "hen", "hero", "hidden", "high", "hill", "hint", "hip", "hire", "history", "hobby",
    "hockey", "hold", "hole", "holiday", "hollow", "home", "honey", "hood", "hope", "horn",
    "horror", "horse", "hospital", "host", "hotel", "hour", "hover", "hub", "huge", "human", "humble",
    "humor", "hundred", "hungry", "hunt", "hurdle", "hurry", "hurt", "husband", "hybrid", "ice",
    "icon", "idea", "identify", "idle", "ignore", "ill", "illegal", "illness", "image", "imitate",
    "immense", "immune", "impact", "impose", "improve", "impulse", "inch", "include", "income",
    "increase", "index", "indicate", "indoor", "industry", "infant", "inflict", "inform", "inhale",
    "inherit", "initial", "inject", "injury", "inmate", "inner", "innocent", "input", "inquiry",
    "insane", "insect", "inside", "inspire", "install", "intact", "interest", "into", "invest",
    "invite", "involve", "iron", "island", "isolate", "issue", "item", "ivory", "jacket", "jaguar",
    "jar", "jazz", "jealous", "jeans", "jelly", "jewel", "job", "join", "joke", "journey", "joy",
    "judge", "juice", "jump", "jungle", "junior", "junk", "just", "kangaroo", "keen", "keep",
    "ketchup", "key", "kick", "kid", "kidney", "kind", "kingdom", "kiss", "kit", "kitchen", "kite",
    "kitten", "kiwi", "knee", "knife", "knock", "know", "lab", "label", "labor", "ladder", "lady",
    "lake", "lamp", "language", "laptop", "large", "later", "latin", "laugh", "laundry", "lava",
    "law", "lawn", "lawsuit", "layer", "lazy", "leader", "leaf", "learn", "leave", "lecture", "left",
    "leg", "legal", "legend", "leisure", "lemon", "lend", "length", "lens", "leopard", "lesson",
    "letter", "level", "liar", "liberty", "library", "license", "life", "lift", "light", "like",
    "limb", "limit", "link", "lion", "liquid", "list", "little", "live", "lizard", "load", "loan",
    "lobster", "local", "lock", "logic", "lonely", "long", "loop", "lottery", "loud", "lounge",
    "love", "loyal", "lucky", "luggage", "lumber", "lunar", "lunch", "luxury", "lyrics", "machine",
    "mad", "magic", "magnet", "maid", "mail", "main", "major", "make", "mammal", "man", "manage",
    "mandate", "mango", "mansion", "manual", "maple", "marble", "march", "margin", "marine",
    "market", "marriage", "mask", "mass", "master", "match", "material", "math", "matrix", "matter",
    "maximum", "maze", "meadow", "mean", "measure", "meat", "mechanic", "medal", "media", "melody",
    "melt", "member", "memory", "mention", "menu", "mercy", "merge", "merit", "merry", "mesh",
    "message", "metal", "method", "middle", "midnight", "milk", "million", "mimic", "mind",
    "minimum", "minor", "minute", "miracle", "mirror", "misery", "miss", "mistake", "mix", "mixed",
    "mixture", "mobile", "model", "modify", "mom", "moment", "monitor", "monkey", "monster",
    "month", "moon", "moral", "more", "morning", "mosquito", "mother", "motion", "motor",
    "mountain", "mouse", "move", "movie", "much", "muffin", "mule", "multiply", "muscle", "museum",
    "mushroom", "music", "must", "mutual", "myself", "mystery", "myth", "naive", "name", "napkin",
    "narrow", "nasty", "nation", "nature", "near", "neck", "need", "negative", "neglect", "neither",
    "nephew", "nerve", "nest", "net", "network", "neutral", "never", "news", "next", "nice", "night",
    "noble", "noise", "nominee", "noodle", "normal", "north", "nose", "notable", "note", "nothing",
    "notice", "novel", "now", "nuclear", "number", "nurse", "nut", "oak", "obey", "object",
    "oblige", "obscure", "observe", "obtain", "obvious", "occur", "ocean", "october", "odor",
    "off", "offer", "office", "often", "oil", "okay", "old", "olive", "olympic", "omit", "once",
    "one", "onion", "online", "only", "open", "opera", "opinion", "oppose", "option", "orange",
    "orbit", "orchard", "order", "ordinary", "organ", "orient", "original", "orphan", "ostrich",
    "other", "outdoor", "outer", "output", "outside", "oval", "oven", "over", "own", "owner",
    "oxygen", "oyster", "ozone", "pact", "paddle", "page", "pair", "palace", "palm", "panda",
    "panel", "panic", "panther", "paper", "parade", "parent", "park", "parrot", "party", "pass",
    "patch", "path", "patient", "patrol", "pattern", "pause", "pave", "payment", "peace", "peanut",
    "pear", "peasant", "pelican", "pen", "penalty", "pencil", "people", "pepper", "perfect",
    "permit", "person", "pet", "phone", "photo", "phrase", "physical", "piano", "picnic",
    "picture", "piece", "pig", "pigeon", "pill", "pilot", "pink", "pioneer", "pipe", "pistol",
    "pitch", "pizza", "place", "planet", "plastic", "plate", "play", "please", "pledge", "pluck",
    "plug", "plunge", "poem", "poet", "point", "polar", "pole", "police", "pond", "pony", "pool",
    "popular", "portion", "position", "possible", "post", "potato", "pottery", "poverty", "powder",
    "power", "practice", "praise", "predict", "prefer", "prepare", "present", "pretty", "prevent",
    "price", "pride", "primary", "print", "priority", "prison", "private", "prize", "problem",
    "process", "produce", "profit", "program", "project", "promote", "proof", "property", "prosper",
    "protect", "proud", "provide", "public", "pudding", "pull", "pulp", "pulse", "pumpkin", "punch",
    "pupil", "puppy", "purchase", "purity", "purpose", "purse", "push", "put", "puzzle", "pyramid",
    "quality", "quantum", "quarter", "question", "quick", "quit", "quiz", "quote", "rabbit",
    "raccoon", "race", "rack", "radar", "radio", "rail", "rain", "raise", "rally", "ramp", "ranch",
    "random", "range", "rapid", "rare", "rate", "rather", "raven", "raw", "razor", "ready", "real",
    "reason", "rebel", "rebuild", "recall", "receive", "recipe", "record", "recycle", "reduce",
    "reflect", "reform", "refuse", "region", "regret", "regular", "reject", "relax", "release",
    "relief", "rely", "remain", "remember", "remind", "remove", "render", "renew", "rent",
    "reopen", "repair", "repeat", "replace", "report", "require", "rescue", "resemble", "resist",
    "resource", "response", "result", "retire", "retreat", "return", "reunion", "reveal", "review",
    "reward", "rhythm", "rib", "ribbon", "rice", "rich", "ride", "ridge", "rifle", "right", "rigid",
    "ring", "riot", "ripple", "risk", "ritual", "rival", "river", "road", "roast", "robot", "robust",
    "rocket", "romance", "roof", "rookie", "room", "rose", "rotate", "rough", "round", "route",
    "royal", "rubber", "rude", "rug", "rule", "run", "runway", "rural", "sad", "saddle", "sadness",
    "safe", "sail", "salad", "salmon", "salon", "salt", "salute", "same", "sample", "sand",
    "satisfy", "satoshi", "sauce", "sausage", "save", "say", "scale", "scan", "scare", "scatter",
    "scene", "scheme", "school", "science", "scissors", "scorpion", "scout", "scrap", "screen",
    "script", "scrub", "sea", "search", "season", "seat", "second", "secret", "section", "security",
    "seed", "seek", "segment", "select", "sell", "seminar", "senior", "sense", "sentence", "series",
    "service", "session", "settle", "setup", "seven", "shadow", "shaft", "shallow", "share", "shed",
    "shell", "sheriff", "shield", "shift", "shine", "ship", "shiver", "shock", "shoe", "shoot",
    "shop", "short", "shoulder", "shove", "shrimp", "shrug", "shuffle", "shy", "sibling", "sick",
    "side", "siege", "sight", "sign", "silent", "silk", "silly", "silver", "similar", "simple",
    "since", "sing", "siren", "sister", "situate", "six", "size", "skate", "sketch", "ski", "skill",
    "skin", "skirt", "skull", "slab", "slam", "sleep", "slender", "slice", "slide", "slight", "slim",
    "slogan", "slot", "slow", "slush", "small", "smart", "smile", "smoke", "smooth", "snack", "snake",
    "snap", "sniff", "snow", "soap", "soccer", "social", "sock", "soda", "soft", "solar", "soldier",
    "solid", "solution", "solve", "someone", "song", "soon", "sorry", "sort", "soul", "sound",
    "soup", "source", "south", "space", "spare", "spatial", "spawn", "speak", "special", "speed",
    "spell", "spend", "sphere", "spice", "spider", "spike", "spin", "spirit", "split", "spoil",
    "sponsor", "spoon", "sport", "spot", "spray", "spread", "spring", "spy", "square", "squeeze",
    "squirrel", "stable", "stadium", "staff", "stage", "stairs", "stamp", "stand", "start", "state",
    "stay", "steak", "steel", "stem", "step", "stereo", "stick", "still", "sting", "stock", "stomach",
    "stone", "stool", "story", "stove", "strategy", "street", "strike", "strong", "struggle",
    "student", "stuff", "stumble", "style", "subject", "submit", "subway", "success", "such",
    "sudden", "suffer", "sugar", "suggest", "suit", "summer", "sun", "sunny", "sunset", "super",
    "supply", "supreme", "sure", "surface", "surge", "surprise", "surround", "survey", "suspect",
    "sustain", "swallow", "swamp", "swap", "swarm", "swear", "sweet", "swift", "swim", "swing",
    "switch", "sword", "symbol", "symptom", "syrup", "system", "table", "tackle", "tag", "tail",
    "talent", "talk", "tank", "tape", "target", "task", "taste", "tattoo", "taxi", "teach", "team",
    "tell", "ten", "tenant", "tennis", "tent", "term", "test", "text", "thank", "that", "theme",
    "then", "theory", "there", "they", "thing", "this", "thought", "three", "thrive", "throw",
    "thumb", "thunder", "ticket", "tide", "tiger", "tilt", "timber", "time", "tiny", "tip", "tired",
    "tissue", "title", "toast", "tobacco", "today", "toddler", "toe", "together", "toilet", "token",
    "tomato", "tomorrow", "tone", "tongue", "tonight", "tool", "tooth", "top", "topic", "topple",
    "torch", "tornado", "tortoise", "toss", "total", "tourist", "toward", "tower", "town", "toy",
    "track", "trade", "traffic", "tragic", "train", "transfer", "trap", "trash", "travel", "tray",
    "treat", "tree", "trend", "trial", "tribe", "trick", "trigger", "trim", "trip", "trophy",
    "trouble", "truck", "true", "truly", "trumpet", "trust", "truth", "try", "tube", "tuition",
    "tumble", "tuna", "tunnel", "turkey", "turn", "turtle", "twelve", "twenty", "twice", "twin",
    "twist", "two", "type", "typical", "ugly", "umbrella", "unable", "unaware", "uncle", "uncover",
    "under", "undo", "unfair", "unfold", "unhappy", "uniform", "unique", "unit", "universe",
    "unknown", "unlock", "until", "unusual", "unveil", "update", "upgrade", "uphold", "upon", "upper",
    "upset", "urban", "urge", "usage", "use", "used", "useful", "useless", "usual", "utility",
    "vacant", "vacuum", "vague", "valid", "valley", "valve", "van", "vanish", "vapor", "various",
    "vast", "vault", "vehicle", "velvet", "vendor", "venture", "venue", "verb", "verify", "version",
    "very", "vessel", "veteran", "viable", "vibrant", "vicious", "victory", "video", "view",
    "village", "vintage", "violin", "virtual", "virus", "visa", "visit", "visual", "vital", "vivid",
    "vocal", "voice", "void", "volcano", "volume", "vote", "voyage", "wage", "wagon", "wait",
    "walk", "wall", "walnut", "want", "warfare", "warm", "warrior", "wash", "wasp", "waste",
    "water", "wave", "way", "wealth", "weapon", "wear", "weasel", "weather", "web", "wedding",
    "weekend", "weird", "welcome", "west", "wet", "whale", "what", "wheat", "wheel", "when", "where",
    "whip", "whisper", "wide", "width", "wife", "wild", "will", "win", "window", "wine", "wing",
    "wink", "winner", "winter", "wire", "wisdom", "wise", "wish", "witness", "wolf", "woman",
    "wonder", "wood", "wool", "word", "work", "world", "worry", "worth", "wrap", "wreck", "wrestle",
    "wrist", "write", "wrong", "yard", "year", "yellow", "you", "young", "youth", "zebra", "zero",
    "zone", "zoo",
];

const HARDENED: u32 = 0x8000_0000;

type HmacSha512 = Hmac<Sha512>;

/// Check a BIP-39 mnemonic: every word must be in the dictionary and the final
/// checksum word must match the SHA-256-based checksum of the entropy.
pub fn validate_mnemonic(mnemonic: &str) -> bool {
    let words: Vec<&str> = mnemonic.split_whitespace().collect();
    if !matches!(words.len(), 12 | 15 | 18 | 21 | 24) {
        return false;
    }

    // Map words to 11-bit indices.
    let mut indices = Vec::with_capacity(words.len());
    for w in &words {
        match word_index(w) {
            Some(i) => indices.push(i as u16),
            None => return false,
        }
    }

    // Reconstruct the entropy bit string, verify the checksum bits.
    let total_bits = words.len() * 11;
    let entropy_bits = total_bits * 32 / 33;
    let checksum_bits = total_bits - entropy_bits;

    let mut bits = String::with_capacity(total_bits);
    for i in &indices {
        bits.push_str(&format!("{:011b}", i));
    }

    let entropy = bits_to_bytes(&bits[..entropy_bits]);
    let checksum = bits_to_int(&bits[entropy_bits..]);

    let hash = Sha256::digest(&entropy);
    let expected = (hash[0] as usize >> (8 - checksum_bits)) & ((1 << checksum_bits) - 1);
    checksum == expected
}

/// Convert a BIP-39 mnemonic to a 64-byte BIP-39 seed (PBKDF2-HMAC-SHA512,
/// 2048 rounds, salt = "mnemonic" + optional passphrase).
pub fn mnemonic_to_seed(mnemonic: &str, passphrase: &str) -> [u8; 64] {
    let salt = format!("mnemonic{}", passphrase);
    let mut seed = [0u8; 64];
    pbkdf2_hmac::<Sha512>(mnemonic.as_bytes(), salt.as_bytes(), 2048, &mut seed);
    seed
}

/// Generate a BIP-39 mnemonic for 256-bit entropy (24 words) using OS CSPRNG.
pub fn generate_mnemonic_24() -> Result<String, String> {
    use rand::rngs::OsRng;
    use rand::RngCore;

    let mut entropy = [0u8; 32];
    OsRng.fill_bytes(&mut entropy);

    // Append the first 8 bits of SHA-256(entropy) as the checksum.
    let hash = Sha256::digest(&entropy);
    let mut bits = String::with_capacity(264);
    for b in entropy.iter() {
        bits.push_str(&format!("{:08b}", b));
    }
    bits.push_str(&format!("{:08b}", hash[0]));

    let mut words = Vec::with_capacity(24);
    for chunk in 0..24 {
        let idx = usize::from_str_radix(&bits[chunk * 11..(chunk + 1) * 11], 2)
            .map_err(|e| e.to_string())?;
        words.push(BIP39_ENGLISH[idx].to_string());
    }
    Ok(words.join(" "))
}

/// Derive the secp256k1 private key (32 bytes) at a BIP-32 path from a seed.
/// Implements CKDpriv (HMAC-SHA512) with hardened/non-hardened handling.
pub fn derive_private_key(seed: &[u8; 64], path: &[u32]) -> Result<[u8; 32], String> {
    // Master key: I = HMAC-SHA512(key="Bitcoin seed", data=seed)
    let mut mac = HmacSha512::new_from_slice(b"Bitcoin seed")
        .map_err(|e| e.to_string())?;
    mac.update(seed);
    let i = mac.finalize().into_bytes();
    let mut key = [0u8; 32];
    let mut chain_code = [0u8; 32];
    key.copy_from_slice(&i[..32]);
    chain_code.copy_from_slice(&i[32..]);

    let secp = secp256k1::Secp256k1::new();

    for &index in path {
        let mut mac = HmacSha512::new_from_slice(&chain_code)
            .map_err(|e| e.to_string())?;

        if index >= HARDENED {
            mac.update(&[0u8]);
            mac.update(&key);
        } else {
            // Non-hardened: compressed public key.
            let sk = secp256k1::SecretKey::from_slice(&key)
                .map_err(|e| e.to_string())?;
            let pk = secp256k1::PublicKey::from_secret_key(&secp, &sk);
            mac.update(&pk.serialize());
        }
        mac.update(&index.to_be_bytes());

        let i = mac.finalize().into_bytes();

        // child = (IL + k_par) mod n  (fails if zero or >= order)
        let tweak = secp256k1::Scalar::from_be_bytes(
            i[..32].try_into().map_err(|_| "invalid IL length")?,
        )
        .map_err(|_| "IL out of range")?;
        let sk = secp256k1::SecretKey::from_slice(&key)
            .map_err(|e| e.to_string())?;
        let child = sk.add_tweak(&tweak).map_err(|e| e.to_string())?;
        key = child.secret_bytes();
        chain_code.copy_from_slice(&i[32..]);
    }

    Ok(key)
}

/// Derive the EVM (Ethereum) address from a 32-byte private key:
/// uncompressed pubkey -> Keccak-256 -> last 20 bytes -> 0x-prefixed hex.
pub fn evm_address_from_private_key(priv_key: &[u8; 32]) -> String {
    let secp = secp256k1::Secp256k1::new();
    let sk = secp256k1::SecretKey::from_slice(priv_key)
        .expect("valid private key");
    let pk = secp256k1::PublicKey::from_secret_key(&secp, &sk);
    let serialized = pk.serialize_uncompressed();

    let mut hasher = Keccak256::new();
    hasher.update(&serialized[1..]);
    let hash = hasher.finalize();
    format!("0x{}", hex::encode(&hash[12..]))
}

/// Derive the EVM address for account `index` at the canonical BIP-44
/// m/44'/60'/0'/0/<index> path from a BIP-39 seed.
pub fn evm_address_from_seed(seed: &[u8; 64], account: u32) -> Result<String, String> {
    let path = [44 | HARDENED, 60 | HARDENED, 0 | HARDENED, 0, account];
    let key = derive_private_key(seed, &path)?;
    Ok(evm_address_from_private_key(&key))
}

fn word_index(word: &str) -> Option<usize> {
    BIP39_ENGLISH.binary_search(&word).ok()
}

/// Convert a binary string to a big-endian byte vector.
fn bits_to_bytes(s: &str) -> Vec<u8> {
    let mut out = Vec::with_capacity((s.len() + 7) / 8);
    let bytes = s.as_bytes();
    let mut current: u8 = 0;
    let mut count = 0;
    for &b in bytes {
        current = (current << 1) | if b == b'1' { 1 } else { 0 };
        count += 1;
        if count == 8 {
            out.push(current);
            current = 0;
            count = 0;
        }
    }
    out
}

/// Convert a binary string to an integer value (used for the checksum bits).
fn bits_to_int(s: &str) -> usize {
    let mut value = 0usize;
    for &b in s.as_bytes() {
        value = (value << 1) | if b == b'1' { 1 } else { 0 };
    }
    value
}

#[cfg(test)]
mod tests {
    use super::*;

    const VECTOR_MNEMONIC: &str = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about";

    #[test]
    fn test_validate_mnemonic_valid() {
        assert!(validate_mnemonic(VECTOR_MNEMONIC));
    }

    #[test]
    fn test_validate_mnemonic_invalid_word() {
        assert!(!validate_mnemonic(
            "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon zzzzz"
        ));
    }

    #[test]
    fn test_validate_mnemonic_bad_length() {
        assert!(!validate_mnemonic("abandon ability able"));
    }

    #[test]
    fn test_mnemonic_to_seed_vector() {
        let seed = mnemonic_to_seed(VECTOR_MNEMONIC, "");
        let hex_seed = hex::encode(seed);
        // Canonical BIP-39 test vector.
        assert_eq!(
            hex_seed,
            "5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc19a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4"
        );
    }

    #[test]
    fn test_evm_address_vector() {
        let seed = mnemonic_to_seed(VECTOR_MNEMONIC, "");
        let addr = evm_address_from_seed(&seed, 0).unwrap();
        assert_eq!(addr, "0x9858effd232b4033e47d90003d41ec34ecaeda94");
        let addr1 = evm_address_from_seed(&seed, 1).unwrap();
        assert_eq!(addr1, "0x6fac4d18c912343bf86fa7049364dd4e424ab9c0");
    }

    #[test]
    fn test_generate_mnemonic_24() {
        let m = generate_mnemonic_24().unwrap();
        assert!(validate_mnemonic(&m));
        assert_eq!(m.split_whitespace().count(), 24);
    }
}
