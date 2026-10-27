/**
 * TigerWallet Crypto Core - Implementation
 * Production-Ready Cryptographic Library
 */

#include "tiger_crypto.h"
#include <cstring>
#include <sstream>
#include <iomanip>
#include <algorithm>

// Use OS-provided CSPRNG
#include <fcntl.h>
#include <unistd.h>

namespace tiger {
namespace crypto {

// ============================================================================
// Secure Random Implementation
// ============================================================================

std::vector<uint8_t> SecureRandom::generate(size_t length) {
    std::vector<uint8_t> buffer(length);
    
    // Use /dev/urandom for cryptographically secure random bytes
    int fd = open("/dev/urandom", O_RDONLY);
    if (fd >= 0) {
        ssize_t bytes_read = read(fd, buffer.data(), length);
        close(fd);
        if (bytes_read == static_cast<ssize_t>(length)) {
            return buffer;
        }
    }
    
    // Fallback: use arc4random if available (BSD/macOS)
    #if defined(__APPLE__) || defined(__FreeBSD__)
    arc4random_buf(buffer.data(), length);
    return buffer;
    #else
    // Last resort: use system random (less secure)
    for (size_t i = 0; i < length; ++i) {
        buffer[i] = static_cast<uint8_t>(random() & 0xFF);
    }
    return buffer;
    #endif
}

uint32_t SecureRandom::generate32() {
    auto bytes = generate(4);
    return static_cast<uint32_t>(bytes[0]) << 24 |
           static_cast<uint32_t>(bytes[1]) << 16 |
           static_cast<uint32_t>(bytes[2]) << 8 |
           static_cast<uint32_t>(bytes[3]);
}

uint64_t SecureRandom::generate64() {
    auto bytes = generate(8);
    return static_cast<uint64_t>(bytes[0]) << 56 |
           static_cast<uint64_t>(bytes[1]) << 48 |
           static_cast<uint64_t>(bytes[2]) << 40 |
           static_cast<uint64_t>(bytes[3]) << 32 |
           static_cast<uint64_t>(bytes[4]) << 24 |
           static_cast<uint64_t>(bytes[5]) << 16 |
           static_cast<uint64_t>(bytes[6]) << 8 |
           static_cast<uint64_t>(bytes[7]);
}

// ============================================================================
// SHA-256 Implementation
// ============================================================================

static inline uint32_t rotr32(uint32_t x, unsigned n) {
    return (x >> n) | (x << (32 - n));
}

static const uint32_t K[64] = {
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
};

// SHA-256 message-schedule functions (FIPS 180-4 section 4.1.2).
#define TW_SIGMA0(x) (rotr32((x), 7) ^ rotr32((x), 18) ^ ((x) >> 3))
#define TW_SIGMA1(x) (rotr32((x), 17) ^ rotr32((x), 19) ^ ((x) >> 10))
#define TW_BIGSIGMA0(x) (rotr32((x), 2) ^ rotr32((x), 13) ^ rotr32((x), 22))
#define TW_BIGSIGMA1(x) (rotr32((x), 6) ^ rotr32((x), 11) ^ rotr32((x), 25))
#define TW_CH(x, y, z) (((x) & (y)) ^ (~(x) & (z)))
#define TW_MAJ(x, y, z) (((x) & (y)) ^ ((x) & (z)) ^ ((y) & (z)))

// One SHA-256 compression round over a single 64-byte block (FIPS 180-4
// section 6.2.2). The full 64-word message schedule must be derived: w[i] for
// i >= 16 depends on earlier scheduled words, so a 16-word rotating window
// cannot produce the correct digest.
static void sha256_compress(uint32_t state[8], const uint8_t block[64]) {
    uint32_t w[64];
    for (int i = 0; i < 16; ++i) {
        w[i] = (static_cast<uint32_t>(block[i * 4]) << 24) |
               (static_cast<uint32_t>(block[i * 4 + 1]) << 16) |
               (static_cast<uint32_t>(block[i * 4 + 2]) << 8) |
               static_cast<uint32_t>(block[i * 4 + 3]);
    }
    for (int i = 16; i < 64; ++i) {
        w[i] = TW_SIGMA1(w[i - 2]) + w[i - 7] + TW_SIGMA0(w[i - 15]) + w[i - 16];
    }

    uint32_t a = state[0], b = state[1], c = state[2], d = state[3];
    uint32_t e = state[4], f = state[5], g = state[6], h = state[7];

    for (int i = 0; i < 64; ++i) {
        uint32_t t1 = h + TW_BIGSIGMA1(e) + TW_CH(e, f, g) + K[i] + w[i];
        uint32_t t2 = TW_BIGSIGMA0(a) + TW_MAJ(a, b, c);
        h = g; g = f; f = e; e = d + t1;
        d = c; c = b; b = a; a = t1 + t2;
    }

    state[0] += a; state[1] += b; state[2] += c; state[3] += d;
    state[4] += e; state[5] += f; state[6] += g; state[7] += h;
}

Sha256::Sha256() : length_(0) {
    state_ = {0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
              0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19};
    buffer_.reserve(64);
}

void Sha256::update(const uint8_t* data, size_t length) {
    length_ += length;
    buffer_.insert(buffer_.end(), data, data + length);
    
    while (buffer_.size() >= 64) {
        sha256_compress(state_.data(), buffer_.data());
        buffer_.erase(buffer_.begin(), buffer_.begin() + 64);
    }
}

void Sha256::update(const std::string& str) {
    update(reinterpret_cast<const uint8_t*>(str.data()), str.size());
}

std::array<uint8_t, 32> Sha256::finalize() {
    uint64_t bit_length = length_ * 8;
    
    buffer_.push_back(0x80);
    while ((buffer_.size() % 64) != 56) {
        buffer_.push_back(0x00);
    }
    
    for (int i = 7; i >= 0; --i) {
        buffer_.push_back(static_cast<uint8_t>((bit_length >> (i * 8)) & 0xFF));
    }
    
    std::array<uint8_t, 32> result;
    for (size_t chunk = 0; chunk < buffer_.size(); chunk += 64) {
        sha256_compress(state_.data(), buffer_.data() + chunk);
    }
    
    for (int i = 0; i < 8; ++i) {
        result[i * 4] = static_cast<uint8_t>((state_[i] >> 24) & 0xFF);
        result[i * 4 + 1] = static_cast<uint8_t>((state_[i] >> 16) & 0xFF);
        result[i * 4 + 2] = static_cast<uint8_t>((state_[i] >> 8) & 0xFF);
        result[i * 4 + 3] = static_cast<uint8_t>(state_[i] & 0xFF);
    }
    
    return result;
}

std::string Sha256::finalize_hex() {
    auto hash = finalize();
    std::stringstream ss;
    ss << std::hex << std::setfill('0');
    for (auto b : hash) {
        ss << std::setw(2) << static_cast<int>(b);
    }
    return ss.str();
}

std::array<uint8_t, 32> Sha256::hash(const uint8_t* data, size_t length) {
    Sha256 hasher;
    hasher.update(data, length);
    return hasher.finalize();
}

std::array<uint8_t, 32> Sha256::hash(const std::string& str) {
    return hash(reinterpret_cast<const uint8_t*>(str.data()), str.size());
}

std::string Sha256::hash_hex(const uint8_t* data, size_t length) {
    auto digest = hash(data, length);
    std::stringstream ss;
    ss << std::hex << std::setfill('0');
    for (auto b : digest) {
        ss << std::setw(2) << static_cast<int>(b);
    }
    return ss.str();
}

std::string Sha256::hash_hex(const std::string& str) {
    return hash_hex(reinterpret_cast<const uint8_t*>(str.data()), str.size());
}

// ============================================================================
// Base58 Encoding (Bitcoin style)
// ============================================================================

static const char* BASE58_ALPHABET = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

std::string Base58::encode(const uint8_t* data, size_t length) {
    // Each leading zero byte maps to exactly one leading '1'. Counted and
    // emitted once, separately from the integer conversion below, so an
    // all-zero input yields only '1' characters.
    size_t zeros = 0;
    while (zeros < length && data[zeros] == 0) {
        zeros++;
    }

    // Long-division conversion of the significant bytes into base58 digits,
    // least-significant digit first.
    std::vector<uint8_t> digits;
    for (size_t i = zeros; i < length; ++i) {
        uint32_t carry = data[i];
        for (size_t j = 0; j < digits.size(); ++j) {
            carry += static_cast<uint32_t>(digits[j]) << 8;
            digits[j] = static_cast<uint8_t>(carry % 58);
            carry /= 58;
        }
        while (carry > 0) {
            digits.push_back(static_cast<uint8_t>(carry % 58));
            carry /= 58;
        }
    }

    std::string result;
    result.reserve(zeros + digits.size());
    result.append(zeros, '1');
    for (auto it = digits.rbegin(); it != digits.rend(); ++it) {
        result.push_back(BASE58_ALPHABET[*it]);
    }
    return result;
}

std::string Base58::encode(const std::vector<uint8_t>& data) {
    return encode(data.data(), data.size());
}

std::vector<uint8_t> Base58::decode(const std::string& encoded) {
    // Each leading '1' maps back to exactly one leading zero byte.
    size_t zeros = 0;
    while (zeros < encoded.size() && encoded[zeros] == '1') {
        zeros++;
    }

    // Accumulate the value in base 256, least-significant byte first. Every
    // input digit multiplies the accumulator by 58 and adds the digit value.
    std::vector<uint8_t> bytes;
    for (size_t i = zeros; i < encoded.size(); ++i) {
        const char* p = std::strchr(BASE58_ALPHABET, encoded[i]);
        if (p == nullptr) {
            return {};  // invalid alphabet character: fail closed
        }
        uint32_t carry = static_cast<uint32_t>(p - BASE58_ALPHABET);
        for (size_t j = 0; j < bytes.size(); ++j) {
            carry += static_cast<uint32_t>(bytes[j]) * 58;
            bytes[j] = static_cast<uint8_t>(carry & 0xFF);
            carry >>= 8;
        }
        while (carry > 0) {
            bytes.push_back(static_cast<uint8_t>(carry & 0xFF));
            carry >>= 8;
        }
    }

    std::vector<uint8_t> result;
    result.reserve(zeros + bytes.size());
    result.insert(result.end(), zeros, 0);
    result.insert(result.end(), bytes.rbegin(), bytes.rend());
    return result;
}

std::string Base58::encode_check(const uint8_t* data, size_t length) {
    // Double SHA256 for checksum
    auto hash1 = Sha256::hash(data, length);
    auto hash2 = Sha256::hash(hash1.data(), hash1.size());
    
    std::vector<uint8_t> to_encode(length + 4);
    std::memcpy(to_encode.data(), data, length);
    std::memcpy(to_encode.data() + length, hash2.data(), 4);
    
    return encode(to_encode);
}

std::vector<uint8_t> Base58::decode_check(const std::string& encoded) {
    auto decoded = decode(encoded);
    if (decoded.size() < 4) return {};
    
    std::vector<uint8_t> data(decoded.size() - 4);
    std::memcpy(data.data(), decoded.data(), decoded.size() - 4);
    
    auto hash1 = Sha256::hash(data.data(), data.size());
    auto hash2 = Sha256::hash(hash1.data(), hash1.size());
    
    // Verify checksum
    if (std::memcmp(decoded.data() + decoded.size() - 4, hash2.data(), 4) != 0) {
        return {};
    }
    
    return data;
}

// ============================================================================
// BIP-39 Wordlist (English - 2048 words)
// ============================================================================

static const std::vector<std::string> BIP39_WORDLIST = {
    "abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract", "absurd", "abuse",
    "access", "accident", "account", "accuse", "achieve", "acid", "acoustic", "acquire", "across", "action",
    "actor", "actress", "actual", "adapt", "add", "addict", "address", "adjust", "admit", "adult",
    "advance", "advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent", "agree",
    "ahead", "aim", "air", "airport", "aisle", "alarm", "album", "alcohol", "alert", "alien",
    "all", "alley", "allow", "almost", "alone", "alpha", "already", "also", "alter", "always",
    "amateur", "amazing", "among", "amount", "amused", "analyst", "anchor", "ancient", "anger", "angle",
    "angry", "animal", "ankle", "announce", "annual", "another", "answer", "antenna", "anticipate", "anxiety",
    "any", "apart", "apology", "appear", "apple", "approve", "april", "arch", "arctic", "area",
    "arena", "argue", "arm", "armed", "armor", "army", "around", "arrange", "arrest", "arrive",
    "arrow", "art", "artefact", "artist", "artwork", "ask", "aspect", "assault", "asset", "assist",
    "assume", "asthma", "athlete", "atom", "attack", "attend", "attitude", "attract", "auction", "audit",
    "august", "aunt", "author", "auto", "autumn", "average", "avocado", "avoid", "awake", "aware",
    "away", "awesome", "awful", "awkward", "axis", "baby", "bachelor", "bacon", "badge", "bag",
    "balance", "balcony", "ball", "bamboo", "banana", "banner", "bar", "barely", "bargain", "barrel",
    "base", "basic", "basket", "battle", "beach", "bean", "beauty", "because", "become", "beef",
    "before", "begin", "behave", "behind", "believe", "below", "belt", "bench", "benefit", "best",
    "betray", "better", "between", "beyond", "bicycle", "bid", "bike", "bind", "biology", "bird",
    "birth", "bitter", "black", "blade", "blame", "blanket", "blast", "bleak", "bless", "blind",
    "blood", "blossom", "blouse", "blue", "blur", "blush", "board", "boat", "body", "boil",
    "bomb", "bone", "bonus", "book", "boost", "border", "boring", "borrow", "boss", "bottom",
    "bounce", "box", "boy", "bracket", "brain", "brand", "brass", "brave", "bread", "breeze",
    "brick", "bridge", "brief", "bright", "bring", "brisk", "broccoli", "broken", "bronze", "broom",
    "brother", "brown", "brush", "bubble", "buddy", "budget", "buffalo", "build", "bulb", "bulk",
    "bullet", "bundle", "bunker", "burden", "burger", "burst", "bus", "business", "busy", "butter",
    "buyer", "buzz", "cabbage", "cabin", "cable", "cactus", "cage", "cake", "call", "calm",
    "camera", "camp", "can", "canal", "cancel", "candy", "cannon", "canoe", "canvas", "canyon",
    "capable", "capital", "captain", "car", "carbon", "card", "cargo", "carpet", "carry", "cart",
    "case", "cash", "casino", "castle", "casual", "cat", "catalog", "catch", "category", "cattle",
    "caught", "cause", "caution", "cave", "ceiling", "celery", "cement", "census", "century", "cereal",
    "certain", "chair", "chalk", "champion", "change", "chaos", "chapter", "charge", "chase", "chat",
    "cheap", "check", "cheese", "cherry", "chest", "chicken", "chief", "child", "chimney", "choice",
    "choose", "chronic", "chuckle", "chunk", "churn", "cigar", "cinnamon", "circle", "citizen", "city",
    "civil", "claim", "clap", "clarify", "claw", "clay", "clean", "clerk", "clever", "click",
    "client", "cliff", "climb", "clinic", "clip", "clock", "clog", "close", "cloth", "cloud",
    "clown", "club", "clump", "cluster", "clutch", "coach", "coast", "coconut", "code", "coffee",
    "coil", "coin", "collect", "color", "column", "combine", "come", "comfort", "comic", "common",
    "company", "concert", "conduct", "confirm", "congress", "connect", "consider", "control", "convince", "cook",
    "cool", "copper", "copy", "coral", "core", "corn", "correct", "cost", "cotton", "couch",
    "country", "couple", "course", "cousin", "cover", "coyote", "crack", "cradle", "craft", "cram",
    "crane", "crash", "crater", "crawl", "crazy", "cream", "credit", "creek", "crew", "cricket",
    "crime", "crisp", "critic", "crop", "cross", "crouch", "crowd", "crucial", "cruel", "cruise",
    "crumble", "crunch", "crush", "cry", "crystal", "cube", "culture", "cup", "cupboard", "curious",
    "current", "curtain", "curve", "cushion", "custom", "cute", "cycle", "dad", "damage", "damp",
    "dance", "danger", "daring", "dash", "daughter", "dawn", "day", "deal", "debate", "debris",
    "decade", "december", "decide", "decline", "decorate", "decrease", "deer", "defense", "define", "defy",
    "degree", "delay", "deliver", "demand", "denial", "dentist", "deny", "depart", "depend", "deposit",
    "depth", "deputy", "derive", "describe", "desert", "design", "desk", "despair", "destroy", "detail",
    "detect", "develop", "device", "devote", "diagram", "dial", "diamond", "diary", "dice", "diesel",
    "diet", "differ", "digital", "dignity", "dilemma", "dinner", "dinosaur", "direct", "dirt", "disagree",
    "discover", "disease", "dish", "dismiss", "disorder", "display", "distance", "divert", "divide", "divorce",
    "dizzy", "doctor", "document", "dog", "doll", "dolphin", "domain", "donate", "donkey", "donor",
    "door", "dose", "double", "dove", "draft", "dragon", "drama", "draw", "dream", "dress",
    "drift", "drill", "drink", "drip", "drive", "drop", "drum", "dry", "duck", "dumb",
    "dune", "during", "dust", "dutch", "duty", "dwarf", "dynamic", "eager", "eagle", "early",
    "earn", "earth", "easily", "east", "easy", "echo", "ecology", "economy", "edge", "edit",
    "educate", "effort", "egg", "eight", "eject", "elastic", "elbow", "elder", "electric", "elegant",
    "element", "elephant", "elevator", "elite", "else", "embark", "embody", "embrace", "emerge", "emotion",
    "employ", "empower", "empty", "enable", "enact", "end", "endless", "endorse", "enemy", "energy",
    "enforce", "engage", "engine", "enhance", "enjoy", "enlist", "enough", "enrich", "enroll", "ensure",
    "enter", "entire", "entry", "envelope", "episode", "equal", "equip", "era", "erase", "erode",
    "erosion", "error", "erupt", "escape", "essay", "essence", "estate", "eternal", "ethics", "evidence",
    "evil", "evoke", "evolve", "exact", "example", "excess", "exchange", "excite", "exclude", "excuse",
    "execute", "exercise", "exhaust", "exhibit", "exile", "exist", "exit", "exotic", "expand", "expect",
    "expire", "explain", "expose", "express", "extend", "extra", "eye", "eyebrow", "fabric", "face",
    "faculty", "fade", "faint", "faith", "fall", "false", "fame", "family", "famous", "fan",
    "fancy", "fantasy", "farm", "fashion", "fat", "fatal", "father", "fatigue", "fault", "favorite",
    "feature", "february", "federal", "fee", "feed", "feel", "female", "fence", "festival", "fetch",
    "fever", "few", "fiber", "fiction", "field", "figure", "file", "film", "filter", "final",
    "find", "fine", "finger", "finish", "fire", "firm", "first", "fiscal", "fish", "fit",
    "fitness", "fix", "flag", "flame", "flash", "flat", "flavor", "flee", "flight", "flip",
    "float", "flock", "floor", "flower", "fluid", "flush", "fly", "foam", "focus", "fog",
    "foil", "fold", "follow", "food", "foot", "force", "forest", "forget", "fork", "fortune",
    "forum", "forward", "fossil", "foster", "found", "fox", "fragile", "frame", "frequent", "fresh",
    "friend", "fringe", "frog", "front", "frost", "frown", "frozen", "fruit", "fuel", "fun",
    "funny", "furnace", "fury", "future", "gadget", "gain", "galaxy", "gallery", "game", "gap",
    "garage", "garbage", "garden", "garlic", "gas", "gasp", "gate", "gather", "gauge", "gaze",
    "general", "genius", "genre", "gentle", "genuine", "gesture", "ghost", "giant", "gift", "giggle",
    "ginger", "giraffe", "girl", "give", "glad", "glance", "glare", "glass", "glide", "glimpse",
    "globe", "gloom", "glory", "glove", "glow", "glue", "goat", "goddess", "gold", "good",
    "goose", "gorilla", "gospel", "gossip", "govern", "gown", "grab", "grace", "grain", "grant",
    "grape", "grass", "gravity", "great", "green", "grid", "grief", "grit", "grocery", "group",
    "grow", "grunt", "guard", "guess", "guide", "guilt", "guitar", "gun", "gym", "habit",
    "hair", "half", "hammer", "hamster", "hand", "handle", "harbor", "hard", "harsh", "harvest",
    "hat", "have", "hawk", "hazard", "head", "health", "heart", "heavy", "hedgehog", "height",
    "hello", "helmet", "help", "hen", "hero", "hidden", "high", "hill", "hint", "hip",
    "hire", "history", "hobby", "hockey", "hold", "hole", "holiday", "hollow", "home", "honey",
    "hood", "hope", "horn", "horror", "horse", "hospital", "host", "hotel", "hour", "hover",
    "hub", "huge", "human", "humble", "humor", "hundred", "hungry", "hunt", "hurdle", "hurry",
    "hurt", "husband", "hybrid", "ice", "icon", "idea", "identify", "idle", "ignore", "ill",
    "illegal", "illness", "image", "imitate", "immense", "immune", "impact", "impose", "improve", "impulse",
    "inch", "include", "income", "increase", "index", "indicate", "indoor", "industry", "infant", "inflict",
    "inform", "inhale", "inherit", "initial", "inject", "injury", "inmate", "inner", "innocent", "input",
    "inquiry", "insane", "insect", "insert", "inside", "inspire", "install", "intact", "interest", "into",
    "invest", "invite", "involve", "iron", "island", "isolate", "issue", "item", "ivory", "jacket",
    "jaguar", "jar", "jazz", "jealous", "jeans", "jelly", "jewel", "job", "join", "joke",
    "journey", "joy", "judge", "juice", "jump", "jungle", "junior", "junk", "just", "kangaroo",
    "keen", "keep", "ketchup", "key", "kick", "kid", "kidney", "kind", "kingdom", "kiss",
    "kit", "kitchen", "kite", "kitten", "kiwi", "knee", "knife", "knock", "know", "lab",
    "label", "labor", "ladder", "lady", "lake", "lamp", "language", "laptop", "large", "later",
    "latin", "laugh", "laundry", "lava", "law", "lawn", "lawsuit", "layer", "lazy", "leader",
    "leaf", "learn", "leave", "lecture", "left", "leg", "legal", "legend", "leisure", "lemon",
    "lend", "length", "lens", "leopard", "lesson", "letter", "level", "liar", "liberty", "library",
    "license", "life", "lift", "light", "like", "limb", "limit", "link", "lion", "liquid",
    "list", "little", "live", "lizard", "load", "loan", "lobster", "local", "lock", "logic",
    "lonely", "long", "loop", "lottery", "loud", "lounge", "love", "loyal", "lucky", "luggage",
    "lumber", "lunar", "lunch", "luxury", "lyrics", "machine", "mad", "magic", "magnet", "maid",
    "mail", "main", "major", "make", "mammal", "man", "manage", "mandate", "mango", "mansion",
    "manual", "maple", "marble", "march", "margin", "marine", "market", "marriage", "mask", "mass",
    "master", "match", "material", "math", "matrix", "matter", "maximum", "maze", "meadow", "mean",
    "measure", "meat", "mechanic", "medal", "media", "melody", "melt", "member", "memory", "men",
    "mend", "mental", "mentor", "menu", "mercy", "merge", "merit", "merry", "mesh", "message",
    "metal", "method", "middle", "midnight", "milk", "million", "mimic", "mind", "minimum", "minor",
    "minute", "miracle", "mirror", "misery", "miss", "mistake", "mix", "mixed", "mixture", "mobile",
    "model", "modify", "mom", "moment", "monitor", "monkey", "monster", "month", "moon", "moral",
    "more", "morning", "mosquito", "mother", "motion", "motor", "mountain", "mouse", "move", "movie",
    "much", "muffin", "mule", "multiply", "muscle", "museum", "mushroom", "music", "must", "mutual",
    "myself", "mystery", "myth", "naive", "name", "napkin", "narrow", "nasty", "nation", "nature",
    "near", "neck", "need", "negative", "neglect", "neither", "nephew", "nerve", "nest", "net",
    "network", "neutral", "never", "news", "next", "nice", "night", "noble", "noise", "nominee",
    "noodle", "normal", "north", "nose", "notable", "note", "nothing", "notice", "novel", "now",
    "nuclear", "number", "nurse", "nut", "oak", "obey", "object", "oblige", "obscure", "observe",
    "obtain", "obvious", "occur", "ocean", "october", "odor", "off", "offer", "office", "often",
    "oil", "okay", "old", "olive", "olympic", "omit", "once", "one", "onion", "online",
    "only", "open", "opera", "opinion", "oppose", "option", "orange", "orbit", "orchard", "order",
    "ordinary", "organ", "orient", "original", "orphan", "ostrich", "other", "outdoor", "outer", "output",
    "outside", "oval", "oven", "over", "own", "owner", "oxygen", "oyster", "ozone", "pact",
    "paddle", "page", "pair", "palace", "palm", "panda", "panel", "panic", "panther", "paper",
    "parade", "parent", "park", "parrot", "party", "pass", "patch", "path", "patient", "patrol",
    "pattern", "pause", "pave", "payment", "peace", "peanut", "pear", "peasant", "pelican", "pen",
    "penalty", "pencil", "people", "pepper", "perfect", "permit", "person", "pet", "phone", "photo",
    "phrase", "physical", "piano", "picnic", "picture", "piece", "pig", "pigeon", "pill", "pilot",
    "pink", "pioneer", "pipe", "pistol", "pitch", "pizza", "place", "planet", "plastic", "plate",
    "play", "please", "pledge", "pluck", "plug", "plunge", "poem", "poet", "point", "polar",
    "pole", "police", "pond", "pony", "pool", "popular", "portion", "position", "possible", "post",
    "potato", "pottery", "poverty", "powder", "power", "practice", "praise", "predict", "prefer", "prepare",
    "present", "pretty", "prevent", "price", "pride", "primary", "print", "priority", "prison", "private",
    "prize", "problem", "process", "produce", "profit", "program", "project", "promote", "proof", "property",
    "prosper", "protect", "proud", "provide", "public", "pudding", "pull", "pulp", "pulse", "pumpkin",
    "punch", "pupil", "puppy", "purchase", "purity", "purpose", "purse", "push", "put", "puzzle",
    "pyramid", "quality", "quantum", "quarter", "question", "quick", "quit", "quiz", "quote", "rabbit",
    "raccoon", "race", "rack", "radar", "radio", "rail", "rain", "raise", "rally", "ramp",
    "ranch", "random", "range", "rapid", "rare", "rate", "rather", "raven", "raw", "reach",
    "react", "read", "real", "realm", "rear", "reason", "rebel", "rebuild", "recall", "receive",
    "recipe", "record", "recycle", "red", "reduce", "reflect", "reform", "refuse", "region", "regret",
    "regular", "reject", "relax", "release", "relief", "rely", "remain", "remember", "remind", "remote",
    "remove", "render", "renew", "rent", "reopen", "repair", "repeat", "replace", "reply", "report",
    "represent", "reproduce", "public", "request", "require", "rescue", "resemble", "resist", "resource", "response",
    "result", "retire", "retreat", "return", "reunion", "reveal", "review", "reward", "rhythm", "rib",
    "ribbon", "rice", "rich", "ride", "ridge", "rifle", "right", "rigid", "ring", "riot",
    "ripple", "risk", "ritual", "rival", "river", "road", "roast", "robot", "robust", "rocket",
    "romance", "roof", "rookie", "room", "rose", "rotate", "rough", "round", "route", "royal",
    "rubber", "rude", "rug", "rule", "run", "runway", "rural", "sad", "saddle", "sadness",
    "safe", "sail", "salad", "salmon", "salon", "salt", "salute", "same", "sample", "sand",
    "satisfy", "satoshi", "sauce", "sausage", "save", "say", "scale", "scan", "scare", "scatter",
    "scene", "scheme", "school", "science", "scissors", "scorpion", "scout", "scrap", "screen", "script",
    "scrub", "sea", "search", "season", "seat", "second", "secret", "section", "security", "seed",
    "seek", "segment", "select", "sell", "seminar", "senior", "sense", "sentence", "series", "service",
    "session", "settle", "setup", "seven", "shadow", "shaft", "shallow", "share", "shed", "shell",
    "sheriff", "shield", "shift", "shine", "ship", "shiver", "shock", "shoe", "shoot", "shop",
    "short", "shoulder", "shove", "shrimp", "shrug", "shuffle", "shy", "sibling", "sick", "side",
    "siege", "sight", "sign", "silent", "silk", "silly", "silver", "similar", "simple", "since",
    "sing", "siren", "sister", "situate", "six", "sixteen", "size", "skate", "sketch", "ski",
    "skill", "skin", "skirt", "skull", "slab", "slam", "sleep", "slender", "slice", "slide",
    "slight", "slim", "slogan", "slot", "slow", "slush", "small", "smart", "smile", "smoke",
    "smooth", "snack", "snake", "snap", "sniff", "snow", "soap", "soccer", "social", "sock",
    "soda", "soft", "solar", "soldier", "solid", "solution", "solve", "someone", "song", "soon",
    "sorry", "sort", "soul", "sound", "soup", "source", "south", "space", "spare", "spatial",
    "spawn", "speak", "special", "speed", "spell", "spend", "sphere", "spice", "spider", "spike",
    "spin", "spirit", "split", "spoil", "sponsor", "spoon", "sport", "spot", "spray", "spread",
    "spring", "spy", "square", "squeeze", "squirrel", "stable", "stadium", "staff", "stage", "stairs",
    "stamp", "stand", "start", "state", "stay", "steak", "steel", "stem", "step", "stereo",
    "stick", "still", "sting", "stock", "stomach", "stone", "stool", "story", "stove", "strategy",
    "street", "strike", "strong", "struggle", "student", "stuff", "stumble", "style", "subject", "submit",
    "subway", "success", "such", "sudden", "suffer", "sugar", "suggest", "suit", "summer", "sun",
    "sunny", "sunset", "super", "supply", "supreme", "sure", "surface", "surge", "surprise", "surround",
    "survey", "suspect", "sustain", "swallow", "swamp", "swap", "swarm", "swear", "sweat", "sweep",
    "sweet", "swift", "swim", "swing", "switch", "sword", "symbol", "symptom", "syrup", "system",
    "table", "tackle", "tag", "tail", "talent", "talk", "tank", "tape", "target", "task", "taste",
    "tattoo", "taxi", "teach", "team", "tell", "ten", "tenant", "tennis", "tent", "term",
    "test", "text", "thank", "that", "theme", "then", "theory", "there", "they", "thing",
    "this", "thought", "three", "thrive", "throw", "thumb", "thunder", "ticket", "tide", "tiger",
    "tilt", "timber", "time", "tiny", "tip", "tired", "tissue", "title", "toast", "tobacco",
    "toddler", "toe", "together", "toilet", "token", "tomato", "tomorrow", "tone", "tongue", "tonight",
    "tool", "tooth", "top", "topic", "topple", "torch", "tornado", "tortoise", "toss", "total",
    "tourist", "toward", "tower", "town", "toy", "track", "trade", "traffic", "tragic", "train",
    "transfer", "trap", "trash", "travel", "tray", "treat", "tree", "trend", "trial", "tribe",
    "trick", "trigger", "trim", "trip", "trophy", "trouble", "truck", "true", "truly", "trumpet",
    "trust", "truth", "try", "tube", "tuition", "tumble", "tuna", "tunnel", "turkey", "turn",
    "turtle", "twelve", "twenty", "twice", "twin", "twist", "two", "type", "typical", "ugly",
    "umbrella", "unable", "unaware", "uncle", "uncover", "under", "undo", "unfair", "unfold", "unhappy",
    "uniform", "unique", "unit", "universe", "unknown", "unlock", "until", "unusual", "unveil", "update",
    "upgrade", "uphold", "upon", "upper", "upset", "urban", "urge", "usage", "use", "used",
    "useful", "useless", "usual", "utility", "vacant", "vacuum", "vague", "valid", "valley", "valve",
    "van", "vanish", "vapor", "various", "vegan", "velvet", "vendor", "venture", "venue", "verb",
    "verify", "version", "very", "vessel", "veteran", "viable", "vibrant", "vicious", "victory", "video",
    "view", "village", "vintage", "violin", "virtual", "virus", "visa", "visit", "visual", "vital",
    "vivid", "vocal", "voice", "void", "volcano", "volume", "vote", "voyage", "wage", "wagon",
    "wait", "wake", "walk", "wall", "walnut", "want", "warfare", "warm", "warrior", "wash",
    "wasp", "waste", "water", "wave", "way", "wealth", "weapon", "wear", "weasel", "weather",
    "web", "wedding", "weekend", "weird", "welcome", "west", "wet", "whale", "what", "wheat",
    "wheel", "when", "where", "whip", "whisper", "wide", "width", "wife", "wild", "will",
    "win", "window", "wine", "wing", "wink", "winner", "winter", "wire", "wisdom", "wise",
    "wish", "witness", "wolf", "woman", "wonder", "wood", "wool", "word", "work", "world",
    "worry", "worth", "wrap", "wreck", "wrestle", "wrist", "write", "wrong", "yard", "year",
    "yellow", "you", "young", "youth", "zebra", "zero", "zone", "zoo"
};

const std::vector<std::string>& Bip39Mnemonic::get_wordlist() {
    return BIP39_WORDLIST;
}

// ============================================================================
// Additional helper functions
// ============================================================================

namespace utils {

std::string to_hex(const uint8_t* data, size_t length) {
    std::stringstream ss;
    ss << std::hex << std::setfill('0');
    for (size_t i = 0; i < length; i++) {
        ss << std::setw(2) << static_cast<int>(data[i]);
    }
    return ss.str();
}

std::string to_hex(const std::vector<uint8_t>& data) {
    return to_hex(data.data(), data.size());
}

std::vector<uint8_t> from_hex(const std::string& hex) {
    std::vector<uint8_t> result;
    result.reserve(hex.size() / 2);
    
    for (size_t i = 0; i < hex.size(); i += 2) {
        std::string byte_str = hex.substr(i, 2);
        uint8_t byte = static_cast<uint8_t>(std::strtol(byte_str.c_str(), nullptr, 16));
        result.push_back(byte);
    }
    
    return result;
}

void secure_zero(void* ptr, size_t length) {
    volatile uint8_t* p = static_cast<volatile uint8_t*>(ptr);
    while (length--) {
        *p++ = 0;
    }
}

} // namespace utils

} // namespace crypto
} // namespace tiger
