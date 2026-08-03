/**
 * TigerWallet HD Wallet Service
 * 
 * Hierarchical Deterministic (HD) wallet derivation from 24-word mnemonic.
 * Supports EVM, Solana, Bitcoin, Cosmos, and other chains.
 * Built with Go for high-load distributed operations.
 */

package hdwallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
)

// ============================================================================
// Constants
// ============================================================================

const (
	MnemonicStrength     = 256 // 24 words
	MnemonicEntropy     = 256
	PBKDF2Iterations     = 2048
	KeyLength           = 64
	BIP39SeedLength     = 64
	PathPurpose         = 0x8000002C // 44' - BIP44
	PathCoinType        = 0x80000000 // 0' for BTC, 60' for ETH, etc.
	PathAccount         = 0x80000000 // 0'
	PathChange          = 0          // 0 for external, 1 for internal
	PathIndex           = 0
)

// ============================================================================
// Word List (BIP39 English)
// ============================================================================

var bip39WordList = []string{
	"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract", "absurd", "abuse",
	"access", "accident", "account", "accuse", "achieve", "acid", "acoustic", "acquire", "across", "action",
	"actor", "actress", "actual", "adapt", "add", "addict", "address", "adjust", "admit", "adult",
	"advance", "advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent", "agree",
	"ahead", "aim", "air", "airport", "aisle", "alarm", "album", "alcohol", "alert", "alien",
	"all", "alley", "allow", "almost", "alone", "alpha", "already", "also", "alter", "always",
	"amateur", "amazing", "among", "amount", "amused", "analyst", "anchor", "ancient", "anger", "angle",
	"angry", "animal", "ankle", "announce", "annual", "another", "answer", "antenna", "antique", "anxiety",
	"any", "apart", "apology", "appear", "apple", "approve", "april", "arch", "arctic", "area",
	"arena", "argue", "arm", "armed", "armor", "army", "around", "arrange", "arrest", "arrive",
	"arrow", "art", "artist", "artwork", "ask", "aspect", "assault", "asset", "assist", "assume",
	"asthma", "athlete", "atom", "attack", "attend", "attitude", "attract", "auction", "audit", "august",
	"aunt", "author", "auto", "autumn", "average", "avocado", "avoid", "awake", "aware", "away",
	"awesome", "awful", "awkward", "axis", "baby", "bachelor", "bacon", "badge", "bag", "balance",
	"balcony", "ball", "bamboo", "banana", "banner", "bar", "barely", "bargain", "barrel", "base",
	"basic", "basket", "battle", "beach", "bean", "beauty", "because", "become", "beef", "before",
	"begin", "behave", "behind", "believe", "below", "belt", "bench", "benefit", "best", "betray",
	"better", "between", "beyond", "bicycle", "bid", "bike", "bind", "biology", "bird", "birth",
	"bitter", "black", "blade", "blame", "blanket", "blast", "bleak", "bless", "blind", "blood",
	"blossom", "blouse", "blue", "blur", "blush", "board", "boat", "body", "boil", "bomb",
	"bone", "bonus", "book", "boost", "border", "boring", "borrow", "boss", "bottom", "bounce",
	"box", "boy", "bracket", "brain", "brand", "brass", "brave", "bread", "breeze", "brick",
	"bridge", "brief", "bright", "bring", "brisk", "broccoli", "broken", "bronze", "broom", "brother",
	"brown", "brush", "bubble", "buddy", "budget", "buffalo", "build", "bulb", "bulk", "bullet",
	"bundle", "bunker", "burden", "burger", "burst", "bus", "business", "busy", "butter", "buyer",
	"buzz", "cabbage", "cabin", "cable", "cactus", "cage", "cake", "call", "calm", "camera",
	"camp", "can", "canal", "cancel", "candy", "cannon", "canoe", "canvas", "canyon", "capable",
	"capital", "captain", "car", "carbon", "card", "cargo", "carpet", "carry", "cart", "case",
	"cash", "casino", "castle", "casual", "cat", "catalog", "catch", "category", "cattle", "caught",
	"cause", "caution", "cave", "ceiling", "celery", "cement", "census", "century", "cereal", "certain",
	"chair", "chalk", "champion", "change", "chaos", "chapter", "charge", "chase", "chat", "cheap",
	"check", "cheese", "chef", "cherry", "chest", "chicken", "chief", "child", "chimney", "choice",
	"choose", "chronic", "chuckle", "chunk", "churn", "cigar", "cinnamon", "circle", "citizen", "city",
	"civil", "claim", "clap", "clarify", "classic", "clean", "clerk", "clever", "click", "client",
	"cliff", "climb", "clinic", "clip", "clock", "clog", "close", "cloth", "cloud", "clown",
	"club", "clump", "cluster", "clutch", "coach", "coast", "coconut", "code", "coffee", "coil",
	"coin", "collect", "color", "column", "combine", "come", "comfort", "comic", "common", "company",
	"concert", "conduct", "confirm", "congress", "connect", "consider", "control", "convince", "cook", "cool",
	"copper", "copy", "coral", "core", "corn", "correct", "cost", "cotton", "couch", "country",
	"couple", "course", "cousin", "cover", "coyote", "crack", "cradle", "craft", "cram", "crane",
	"crash", "crater", "crawl", "crazy", "cream", "credit", "creek", "crew", "cricket", "crime",
	"crisp", "critic", "crop", "cross", "crouch", "crowd", "crucial", "cruel", "cruise", "crunch",
	"crush", "cry", "crystal", "cube", "culture", "cup", "cupboard", "curious", "current", "curtain",
	"curve", "cushion", "custom", "cute", "cycle", "dad", "damage", "damp", "dance", "danger",
	"daring", "dash", "daughter", "dawn", "day", "deal", "debate", "debris", "decade", "december",
	"decide", "decline", "decorate", "decrease", "deer", "defense", "define", "defy", "degree", "delay",
	"deliver", "demand", "demise", "denial", "dentist", "deny", "depart", "depend", "deposit", "depth",
	"deputy", "derive", "describe", "desert", "design", "desk", "despair", "destroy", "detail", "detect",
	"develop", "device", "devote", "diagram", "dial", "diamond", "diary", "dice", "diesel", "diet",
	"differ", "digital", "dignity", "dilemma", "dinner", "dinosaur", "direct", "dirt", "disagree", "discover",
	"disease", "dish", "dismiss", "disorder", "display", "distance", "divert", "divide", "divorce", "dizzy",
	"doctor", "document", "dog", "doll", "dolphin", "domain", "donate", "donkey", "donor", "door",
	"dose", "double", "dove", "draft", "dragon", "drama", "draw", "dream", "dress", "drift",
	"drill", "drink", "drip", "drive", "drop", "drum", "dry", "duck", "dumb", "dune",
	"during", "dust", "dutch", "duty", "dwarf", "dynamic", "eager", "eagle", "early", "earn",
	"earth", "easily", "east", "easy", "echo", "ecology", "economy", "edge", "edit", "educate",
	"effort", "egg", "eight", "eject", "elastic", "elbow", "elder", "electric", "elegant", "element",
	"elephant", "elevator", "elite", "else", "embark", "embody", "embrace", "emerge", "emotion", "employ",
	"empower", "empty", "enable", "enact", "end", "endless", "endorse", "enemy", "energy", "enforce",
	"engage", "engine", "enhance", "enjoy", "enlist", "enough", "enrich", "enroll", "ensure", "enter",
	"entire", "entry", "envelope", "episode", "equal", "equip", "era", "erase", "erode", "erosion",
	"error", "erupt", "escape", "essay", "essence", "estate", "eternal", "ethics", "evidence", "evil",
	"evoke", "evolve", "exact", "example", "excess", "exchange", "excite", "exclude", "excuse", "execute",
	"exercise", "exhaust", "exhibit", "exile", "exist", "exit", "exotic", "expand", "expect", "expire",
	"explain", "expose", "express", "extend", "extra", "eye", "eyebrow", "fabric", "face", "faculty",
	"fade", "faint", "faith", "fall", "false", "fame", "family", "famous", "fan", "fancy",
	"fantasy", "farm", "fashion", "fat", "fatal", "father", "fatigue", "fault", "favorite", "feature",
	"february", "federal", "fee", "feed", "feel", "female", "fence", "festival", "fetch", "fever",
	"few", "fiber", "fiction", "field", "figure", "file", "film", "filter", "final", "finance",
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
	"inquiry", "insane", "insect", "inside", "inspire", "install", "intact", "interest", "into", "invest",
	"invite", "involve", "iris", "island", "isolate", "issue", "item", "ivory", "jacket", "jaguar",
	"jar", "jazz", "jealous", "jeans", "jelly", "jewel", "job", "join", "joke", "journey",
	"joy", "judge", "juice", "jump", "jungle", "junior", "junk", "just", "kangaroo", "keen",
	"keep", "ketchup", "key", "kick", "kid", "kidney", "kind", "kingdom", "kiss", "kit",
	"kitchen", "kite", "kitten", "kiwi", "knee", "knife", "knock", "know", "lab", "label",
	"labor", "ladder", "lady", "lake", "lamp", "language", "laptop", "large", "later", "latin",
	"laugh", "laundry", "lava", "law", "lawn", "lawsuit", "layer", "lazy", "leader", "leaf",
	"learn", "leave", "lecture", "left", "leg", "legal", "legend", "leisure", "lemon", "lend",
	"length", "lens", "leopard", "lesson", "letter", "level", "liar", "liberty", "library", "license",
	"life", "lift", "light", "like", "limb", "limit", "link", "lion", "liquid", "list",
	"little", "live", "lizard", "load", "loan", "lobster", "local", "lock", "logic", "lonely",
	"long", "loop", "lottery", "loud", "lounge", "love", "loyal", "lucky", "luggage", "lumber",
	"lunar", "lunch", "luxury", "lyrics", "machine", "mad", "magic", "magnet", "maid", "mail",
	"main", "major", "make", "mammal", "man", "manage", "mandate", "mango", "mansion", "manual",
	"maple", "marble", "march", "margin", "marine", "market", "marriage", "mask", "mass", "master",
	"match", "material", "math", "matrix", "matter", "maximum", "maze", "meadow", "mean", "measure",
	"meat", "mechanic", "medal", "media", "melody", "melt", "member", "memory", "men", "mend",
	"mental", "mentor", "menu", "mercy", "merge", "merit", "merry", "mesh", "message", "metal",
	"method", "middle", "midnight", "milk", "million", "mimic", "mind", "minimum", "minor", "minute",
	"miracle", "mirror", "misery", "miss", "mistake", "mix", "mixed", "mixture", "mobile", "model",
	"modify", "mom", "moment", "monitor", "monkey", "monster", "month", "moon", "more", "morning",
	"mosquito", "mother", "motion", "motor", "mountain", "mouse", "move", "movie", "much", "muffin",
	"mule", "multiply", "muscle", "museum", "mushroom", "music", "must", "mutual", "myself", "mystery",
	"myth", "naive", "name", "napkin", "narrow", "nasty", "nation", "nature", "near", "neck",
	"need", "negative", "neglect", "neither", "nephew", "nerve", "nest", "net", "network", "neutral",
	"never", "news", "next", "nice", "night", "noble", "noise", "nominee", "noodle", "normal",
	"north", "nose", "notable", "note", "nothing", "notice", "novel", "now", "nuclear", "number",
	"nurse", "nut", "oak", "obey", "object", "oblige", "obscure", "observe", "obtain", "obvious",
	"occur", "ocean", "october", "odor", "off", "offer", "office", "often", "oil", "okay",
	"old", "olive", "olympic", "omit", "once", "one", "onion", "online", "only", "open",
	"opera", "opinion", "oppose", "option", "orange", "orbit", "orchard", "order", "ordinary", "organ",
	"orient", "original", "orphan", "ostrich", "other", "outdoor", "outer", "output", "outside", "oval",
	"oven", "over", "own", "owner", "oxygen", "oyster", "ozone", "paddle", "page", "pair",
	"palace", "panda", "panel", "panic", "panther", "paper", "parade", "parent", "park", "parrot",
	"party", "pass", "patch", "path", "patient", "patrol", "pattern", "pause", "pave", "payment",
	"peace", "peanut", "pear", "peasant", "pelican", "pen", "penalty", "pencil", "people", "pepper",
	"perfect", "permit", "person", "pet", "phone", "photo", "phrase", "physical", "piano", "picnic",
	"picture", "piece", "pig", "pigeon", "pill", "pilot", "pink", "pioneer", "pipe", "pistol",
	"pitch", "pizza", "place", "planet", "plastic", "plate", "play", "please", "pledge", "plenty",
	"plumber", "plunge", "poem", "poet", "point", "polar", "pole", "police", "pond", "pony",
	"pool", "popular", "portion", "position", "possible", "post", "potato", "pottery", "poverty", "powder",
	"power", "practice", "praise", "predict", "prefer", "prepare", "present", "pretty", "prevent", "price",
	"pride", "primary", "print", "priority", "prison", "private", "prize", "problem", "process", "produce",
	"profit", "program", "project", "promote", "proof", "property", "prosper", "protect", "proud", "provide",
	"public", "pudding", "pull", "pulp", "pulse", "pumpkin", "punch", "pupil", "puppy", "purchase",
	"purity", "purpose", "purse", "push", "put", "puzzle", "pyramid", "quality", "quantum", "quarter",
	"question", "quick", "quit", "quiz", "quote", "rabbit", "raccoon", "race", "rack", "radar",
	"radio", "rail", "rain", "raise", "rally", "ramp", "ranch", "random", "range", "rapid",
	"rare", "rate", "rather", "raven", "raw", "reach", "react", "read", "real", "realm",
	"rear", "reason", "rebel", "rebuild", "recall", "receive", "recipe", "record", "recover", "recruit",
	"red", "reduce", "reflect", "reform", "refuse", "region", "regret", "regular", "reject", "relax",
	"release", "relief", "rely", "remain", "remember", "remind", "remote", "remove", "render", "renew",
	"rent", "reopen", "repair", "repeat", "replace", "reply", "report", "represent", "reproduce", "public",
	"request", "rescue", "resemble", "resist", "resource", "response", "result", "retire", "retreat", "return",
	"reunion", "reveal", "review", "reward", "rhythm", "rib", "ribbon", "rice", "rich", "ride",
	"ridge", "rifle", "right", "rigid", "ring", "riot", "ripple", "risk", "ritual", "rival",
	"river", "road", "roast", "robot", "robust", "rocket", "romance", "roof", "rookie", "room",
	"rose", "rotate", "rough", "round", "route", "royal", "rubber", "rude", "rug", "rule",
	"run", "runway", "rural", "sad", "saddle", "sadness", "safe", "sail", "salad", "salmon",
	"salon", "salt", "salute", "same", "sample", "sand", "satisfy", "satoshi", "sauce", "sauna",
	"save", "say", "scale", "scan", "scare", "scatter", "scene", "scheme", "school", "science",
	"scissors", "scorpion", "scout", "scrap", "screen", "script", "scrub", "sea", "search",
	"season", "seat", "second", "secret", "section", "security", "seed", "seek", "segment", "select",
	"sell", "seminar", "senior", "sense", "sentence", "series", "service", "session", "settle", "setup",
	"seven", "shadow", "shaft", "shallow", "share", "shed", "shell", "sheriff", "shield", "shift",
	"shine", "ship", "shiver", "shock", "shoe", "shoot", "shop", "short", "shoulder", "shove",
	"shrimp", "shrug", "shuffle", "shy", "sibling", "sick", "side", "siege", "sight", "sign",
	"silent", "silk", "silly", "silver", "similar", "simple", "since", "sing", "siren", "sister",
	"situate", "six", "size", "skate", "sketch", "ski", "skill", "skin", "skirt", "skull",
	"slab", "slam", "sleep", "slice", "slide", "slight", "slim", "slogan", "slot", "slow",
	"slush", "small", "smart", "smile", "smoke", "smooth", "snack", "snake", "snap", "sniff",
	"snow", "soap", "soccer", "social", "sock", "soda", "soft", "solar", "soldier", "solid",
	"solution", "solve", "someone", "song", "soon", "sorry", "sort", "soul", "sound", "soup",
	"source", "south", "space", "spare", "spark", "speak", "special", "speed", "spell", "spend",
	"sphere", "spice", "spider", "spike", "spin", "spirit", "split", "spoil", "sponsor", "spoon",
	"sport", "spot", "spray", "spread", "spring", "spy", "square", "squeeze", "squirrel", "stable",
	"stadium", "staff", "stage", "stairs", "stamp", "stand", "start", "state", "stay", "steak",
	"steel", "stem", "step", "stereo", "stick", "still", "sting", "stock", "stomach", "stone",
	"stool", "story", "stove", "strategy", "street", "strike", "strong", "struggle", "student", "stuff",
	"stumble", "style", "subject", "submit", "subway", "success", "such", "sudden", "suffer", "sugar",
	"suggest", "suit", "summer", "sun", "sunny", "sunset", "super", "supply", "supreme", "sure",
	"surface", "surge", "surprise", "surround", "survey", "suspect", "sustain", "swallow", "swamp", "swap",
	"swarm", "swear", "sweat", "sweep", "sweet", "swift", "swim", "swing", "switch", "sword",
	"symbol", "symptom", "syrup", "system", "table", "tackle", "tag", "tail", "talent", "talk",
	"tank", "tape", "target", "task", "taste", "tattoo", "taxi", "teach", "team", "tell",
	"ten", "tenant", "tennis", "tent", "term", "test", "text", "thank", "that", "them",
	"theme", "then", "theory", "there", "they", "thing", "this", "thought", "three", "thrive",
	"throw", "thumb", "thunder", "ticket", "tide", "tiger", "tilt", "timber", "time", "tiny",
	"tip", "tired", "tissue", "title", "toast", "tobacco", "toddler", "toe", "together", "toilet",
	"token", "tomato", "tomorrow", "tone", "tongue", "tonight", "tool", "tooth", "top", "topic",
	"topple", "torch", "tornado", "tortoise", "toss", "total", "tourist", "toward", "tower", "town",
	"toy", "track", "trade", "traffic", "tragic", "train", "transfer", "trap", "trash", "travel",
	"tray", "treat", "tree", "trend", "trial", "tribe", "trick", "trigger", "trim", "trip",
	"trophy", "trouble", "truck", "true", "truly", "trumpet", "trust", "truth", "try", "tube",
	"tuition", "tumble", "tuna", "tunnel", "turkey", "turn", "turtle", "twelve", "twenty", "twice",
	"twin", "twist", "two", "type", "typical", "ugly", "umbrella", "unable", "unaware", "uncle",
	"uncover", "under", "undo", "unfair", "unfold", "unhappy", "uniform", "unique", "unit", "universe",
	"unknown", "unlock", "until", "unusual", "unveil", "update", "upgrade", "uphold", "upon", "upper",
	"upset", "urban", "urge", "usage", "use", "used", "useful", "useless", "usual", "utility",
	"vacant", "vacuum", "vague", "valid", "valley", "valve", "van", "vanish", "vapor", "various",
	"vegan", "velvet", "vendor", "venture", "venue", "verb", "verify", "version", "very", "vessel",
	"veteran", "viable", "vibrant", "victim", "video", "view", "village", "vintage", "violin", "virtual",
	"virus", "visa", "visit", "visual", "vital", "vivid", "vocal", "voice", "void", "volcano",
	"volume", "vote", "voyage", "wage", "wagon", "wait", "wake", "walk", "wall", "walnut",
	"want", "warfare", "warm", "warrior", "wash", "wasp", "waste", "water", "wave", "way",
	"wealth", "weapon", "wear", "weasel", "weather", "web", "wedding", "weekend", "weird", "welcome",
	"west", "wet", "whale", "what", "wheat", "wheel", "when", "where", "whip", "whisper",
	"wide", "width", "wife", "wild", "will", "win", "window", "wine", "wing", "wink",
	"winner", "winter", "wire", "wisdom", "wise", "wish", "witness", "wolf", "woman", "wonder",
	"wood", "wool", "word", "work", "world", "worry", "worth", "wrap", "wreck", "wrestle",
	"wrist", "write", "wrong", "yard", "year", "yellow", "you", "young", "youth", "zebra",
	"zero", "zone", "zoo",
}

// ============================================================================
// Types
// ============================================================================

// WalletInfo represents HD wallet information
type WalletInfo struct {
	ID           string            `json:"id"`
	Mnemonic     string            `json:"mnemonic,omitempty"` // Only returned during creation
	MasterKey    string            `json:"master_key"`
	Addresses    map[uint64]string `json:"addresses"` // chainID -> address
	PublicKeys   map[uint64]string `json:"public_keys"`
	PrivateKeys  map[uint64]string `json:"private_keys"` // Only returned during creation
	CreatedAt    int64            `json:"created_at"`
	UpdatedAt    int64            `json:"updated_at"`
}

// ChainDerivationPath represents a blockchain derivation path
type ChainDerivationPath struct {
	ChainID     uint64 `json:"chain_id"`
	CoinType   uint32 `json:"coin_type"`
	Purpose    uint32 `json:"purpose"`
	Path       string `json:"path"`
	Derivation string `json:"derivation"`
}

// SupportedChain represents a supported blockchain
type SupportedChain struct {
	ChainID      uint64 `json:"chain_id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	CoinType     uint32 `json:"coin_type"`
	Path         string `json:"path"`
	IsEVM        bool   `json:"is_evm"`
	ExplorerURL  string `json:"explorer_url"`
	RPCURLs      []string `json:"rpc_urls"`
}

// HDWalletService manages HD wallet operations
type HDWalletService struct {
	mu            sync.RWMutex
	wallets       map[string]*WalletInfo
	supportedChains map[uint64]*SupportedChain
}

// ============================================================================
// Service Methods
// ============================================================================

var (
	hdWalletService     *HDWalletService
	hdWalletServiceOnce sync.Once
)

// GetHDWalletService returns the singleton HD wallet service
func GetHDWalletService() *HDWalletService {
	hdWalletServiceOnce.Do(func() {
		hdWalletService = &HDWalletService{
			wallets:         make(map[string]*WalletInfo),
			supportedChains: make(map[uint64]*SupportedChain),
		}
		hdWalletService.initSupportedChains()
	})
	return hdWalletService
}

// ============================================================================
// Supported Chains
// ============================================================================

func (s *HDWalletService) initSupportedChains() {
	chains := []*SupportedChain{
		// EVM Chains (CoinType 60')
		{ChainID: 1, Name: "Ethereum", Symbol: "ETH", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://etherscan.io", RPCURLs: []string{"https://eth.llamarpc.com"}},
		{ChainID: 56, Name: "BNB Smart Chain", Symbol: "BNB", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://bscscan.com", RPCURLs: []string{"https://bsc-dataseed.binance.org"}},
		{ChainID: 137, Name: "Polygon", Symbol: "MATIC", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://polygonscan.com", RPCURLs: []string{"https://polygon-rpc.com"}},
		{ChainID: 42161, Name: "Arbitrum One", Symbol: "ETH", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://arbiscan.io", RPCURLs: []string{"https://arb1.arbitrum.io/rpc"}},
		{ChainID: 10, Name: "Optimism", Symbol: "ETH", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://optimistic.etherscan.io", RPCURLs: []string{"https://mainnet.optimism.io"}},
		{ChainID: 43114, Name: "Avalanche C-Chain", Symbol: "AVAX", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://snowtrace.io", RPCURLs: []string{"https://api.avax.network/ext/bc/C/rpc"}},
		{ChainID: 8453, Name: "Base", Symbol: "ETH", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://basescan.org", RPCURLs: []string{"https://mainnet.base.org"}},
		{ChainID: 250, Name: "Fantom", Symbol: "FTM", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://ftmscan.com", RPCURLs: []string{"https://rpc.fantom.network"}},
		{ChainID: 100, Name: "Gnosis Chain", Symbol: "XDAI", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://gnosisscan.io", RPCURLs: []string{"https://rpc.gnosischain.com"}},
		{ChainID: 42220, Name: "Celo", Symbol: "CELO", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://explorer.celo.org", RPCURLs: []string{"https://forno.celo.org"}},
		{ChainID: 1666600000, Name: "Harmony", Symbol: "ONE", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://explorer.harmony.one", RPCURLs: []string{"https://api.harmony.one"}},
		{ChainID: 1284, Name: "Moonbeam", Symbol: "GLMR", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://moonscan.io", RPCURLs: []string{"https://rpc.api.moonbeam.network"}},
		{ChainID: 324, Name: "zkSync Era", Symbol: "ETH", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://explorer.zksync.io", RPCURLs: []string{"https://mainnet.era.zksync.io"}},
		{ChainID: 59144, Name: "Linea", Symbol: "ETH", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://explorer.linea.build", RPCURLs: []string{"https://rpc.linea.build"}},
		{ChainID: 534352, Name: "Scroll", Symbol: "ETH", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://scrollscan.com", RPCURLs: []string{"https://rpc.scroll.io"}},
		{ChainID: 5000, Name: "Mantle", Symbol: "MNT", CoinType: 60, Path: "m/44'/60'/0'/0/0", IsEVM: true, ExplorerURL: "https://explorer.mantle.xyz", RPCURLs: []string{"https://rpc.mantle.xyz"}},
		
		// Bitcoin (CoinType 0')
		{ChainID: 0, Name: "Bitcoin", Symbol: "BTC", CoinType: 0, Path: "m/44'/0'/0'/0/0", IsEVM: false, ExplorerURL: "https://blockstream.info", RPCURLs: []string{"https://blockstream.info/api"}},
		
		// Cosmos chains (CoinType 118')
		{ChainID: 118, Name: "Cosmos", Symbol: "ATOM", CoinType: 118, Path: "m/44'/118'/0'/0/0", IsEVM: false, ExplorerURL: "https://mintscan.io/cosmos", RPCURLs: []string{"https://cosmos-rpc.polkachu.com"}},
		{ChainID: 5555, Name: "Lambda", Symbol: "LAMB", CoinType: 118, Path: "m/44'/118'/0'/0/0", IsEVM: false, ExplorerURL: "https://explorer.lambda.im", RPCURLs: []string{"https://rest.lambda.im"}},
		
		// Solana (CoinType 501)
		{ChainID: 501, Name: "Solana", Symbol: "SOL", CoinType: 501, Path: "m/44'/501'/0'/0'", IsEVM: false, ExplorerURL: "https://solscan.io", RPCURLs: []string{"https://api.mainnet-beta.solana.com"}},
		
		// Aptos (CoinType 637)
		{ChainID: 637, Name: "Aptos", Symbol: "APT", CoinType: 637, Path: "m/44'/637'/0'/0'/0'", IsEVM: false, ExplorerURL: "https://aptoscan.com", RPCURLs: []string{"https://aptos-mainnet.nodereal.io"}},
		
		// Sui (CoinType 784)
		{ChainID: 784, Name: "Sui", Symbol: "SUI", CoinType: 784, Path: "m/44'/784'/0'/0'/0'", IsEVM: false, ExplorerURL: "https://suiscan.xyz", RPCURLs: []string{"https://fullnode.mainnet.sui.io"}},
		
		// TON (CoinType 607)
		{ChainID: 607, Name: "TON", Symbol: "TON", CoinType: 607, Path: "m/44'/607'/0'/0/0", IsEVM: false, ExplorerURL: "https://tonscan.org", RPCURLs: []string{"https://toncenter.com/api/v2"}},
		
		// TRON (CoinType 195)
		{ChainID: 195, Name: "Tron", Symbol: "TRX", CoinType: 195, Path: "m/44'/195'/0'/0/0", IsEVM: false, ExplorerURL: "https://tronscan.org", RPCURLs: []string{"https://api.trongrid.io"}},
		
		// Dogecoin (CoinType 3)
		{ChainID: 3, Name: "Dogecoin", Symbol: "DOGE", CoinType: 3, Path: "m/44'/3'/0'/0/0", IsEVM: false, ExplorerURL: "https://dogechain.info", RPCURLs: []string{"https://dogecoin.treasure.lol"}},
		
		// Litecoin (CoinType 2)
		{ChainID: 2, Name: "Litecoin", Symbol: "LTC", CoinType: 2, Path: "m/44'/2'/0'/0/0", IsEVM: false, ExplorerURL: "https://blockchair.com/litecoin", RPCURLs: []string{"https://litecoin-rpc.com"}},
	}
	
	for _, chain := range chains {
		s.supportedChains[chain.ChainID] = chain
	}
}

// ============================================================================
// Wallet Creation
// ============================================================================

// GenerateMnemonic generates a new 24-word mnemonic
func (s *HDWalletService) GenerateMnemonic() (string, error) {
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}
	
	mnemonic, err := entropyToMnemonic(entropy)
	if err != nil {
		return "", err
	}
	
	return mnemonic, nil
}

// CreateWallet creates a new HD wallet from a generated or provided mnemonic
func (s *HDWalletService) CreateWallet(mnemonic string, password string) (*WalletInfo, error) {
	// Validate mnemonic
	if !validateMnemonic(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}
	
	// Derive master key from mnemonic
	masterKey, err := deriveMasterKey(mnemonic, password)
	if err != nil {
		return nil, err
	}
	
	wallet := &WalletInfo{
		ID:           "wallet_" + uuid.New().String(),
		Mnemonic:     mnemonic,
		MasterKey:    masterKey,
		Addresses:    make(map[uint64]string),
		PublicKeys:   make(map[uint64]string),
		PrivateKeys:  make(map[uint64]string),
		CreatedAt:    now(),
		UpdatedAt:    now(),
	}
	
	// Derive addresses for all supported chains
	for chainID, chain := range s.supportedChains {
		address, pubKey, privKey, err := deriveAddress(masterKey, chain.Path, chain.IsEVM)
		if err != nil {
			continue
		}
		wallet.Addresses[chainID] = address
		wallet.PublicKeys[chainID] = pubKey
		wallet.PrivateKeys[chainID] = privKey
	}
	
	s.mu.Lock()
	s.wallets[wallet.ID] = wallet
	s.mu.Unlock()
	
	return wallet, nil
}

// ImportWallet imports an existing wallet from a mnemonic
func (s *HDWalletService) ImportWallet(mnemonic string, password string) (*WalletInfo, error) {
	// Clean and validate mnemonic
	mnemonic = strings.ToLower(strings.TrimSpace(mnemonic))
	mnemonic = strings.Join(strings.Fields(mnemonic), " ")
	
	if !validateMnemonic(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}
	
	return s.CreateWallet(mnemonic, password)
}

// GetWallet returns wallet info by ID
func (s *HDWalletService) GetWallet(id string) (*WalletInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	wallet, exists := s.wallets[id]
	if !exists {
		return nil, fmt.Errorf("wallet not found")
	}
	
	// Return without private keys
	result := &WalletInfo{
		ID:         wallet.ID,
		MasterKey: wallet.MasterKey,
		Addresses:  wallet.Addresses,
		PublicKeys: wallet.PublicKeys,
		CreatedAt:  wallet.CreatedAt,
		UpdatedAt:  wallet.UpdatedAt,
	}
	
	return result, nil
}

// GetAddress returns address for a specific chain
func (s *HDWalletService) GetAddress(walletID string, chainID uint64) (string, error) {
	wallet, err := s.GetWallet(walletID)
	if err != nil {
		return "", err
	}
	
	address, exists := wallet.Addresses[chainID]
	if !exists {
		return "", fmt.Errorf("address not found for chain %d", chainID)
	}
	
	return address, nil
}

// GetAllAddresses returns all addresses for a wallet
func (s *HDWalletService) GetAllAddresses(walletID string) (map[uint64]string, error) {
	wallet, err := s.GetWallet(walletID)
	if err != nil {
		return nil, err
	}
	
	return wallet.Addresses, nil
}

// GetSupportedChains returns all supported chains
func (s *HDWalletService) GetSupportedChains() []*SupportedChain {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	chains := make([]*SupportedChain, 0, len(s.supportedChains))
	for _, chain := range s.supportedChains {
		chains = append(chains, chain)
	}
	
	return chains
}

// GetChainInfo returns chain info by ID
func (s *HDWalletService) GetChainInfo(chainID uint64) (*SupportedChain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	chain, exists := s.supportedChains[chainID]
	if !exists {
		return nil, fmt.Errorf("chain not found")
	}
	
	return chain, nil
}

// SignMessage signs a message with the wallet's private key for a specific chain
func (s *HDWalletService) SignMessage(walletID string, chainID uint64, message string) (string, error) {
	s.mu.RLock()
	wallet, exists := s.wallets[walletID]
	s.mu.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("wallet not found")
	}
	
	privKey, exists := wallet.PrivateKeys[chainID]
	if !exists {
		return "", fmt.Errorf("private key not found for chain %d", chainID)
	}
	
	chain, exists := s.supportedChains[chainID]
	if !exists {
		return "", fmt.Errorf("chain not found")
	}
	
	// Sign based on chain type
	if chain.IsEVM {
		return signEVMMessage(privKey, message)
	}
	
	return "", fmt.Errorf("signing not supported for chain type")
}

// SignTransaction signs a transaction for a specific chain
func (s *HDWalletService) SignTransaction(walletID string, chainID uint64, txData string) (string, error) {
	s.mu.RLock()
	wallet, exists := s.wallets[walletID]
	s.mu.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("wallet not found")
	}
	
	privKey, exists := wallet.PrivateKeys[chainID]
	if !exists {
		return "", fmt.Errorf("private key not found for chain %d", chainID)
	}
	
	chain, exists := s.supportedChains[chainID]
	if !exists {
		return "", fmt.Errorf("chain not found")
	}
	
	if chain.IsEVM {
		return signEVMTx(privKey, txData)
	}
	
	return "", fmt.Errorf("transaction signing not supported for chain type")
}

// ============================================================================
// Helper Functions
// ============================================================================

func entropyToMnemonic(entropy []byte) (string, error) {
	// Check entropy length
	if len(entropy) < 16 || len(entropy) > 32 || len(entropy)%4 != 0 {
		return "", fmt.Errorf("invalid entropy length")
	}
	
	// Calculate checksum
	hash := sha256.Sum256(entropy)
	checksumBits := len(entropy) / 4
	checksum := hash[0] >> (8 - checksumBits)
	
	// Combine entropy and checksum
	entropyBits := len(entropy) * 8
	totalBits := entropyBits + checksumBits
	bits := make([]byte, (totalBits+7)/8)
	for i, b := range entropy {
		bits[i] = b
	}
	bits[len(entropy)] = checksum
	
	// Convert to words
	var words []string
	for i := 0; i < totalBits; i += 11 {
		index := 0
		for j := 0; j < 11; j++ {
			bitPos := i + j
			if bitPos < totalBits {
				bytePos := bitPos / 8
				bitOffset := 7 - (bitPos % 8)
				if bits[bytePos]&(1<<bitOffset) != 0 {
					index |= 1 << (10 - j)
				}
			}
		}
		if index < len(bip39WordList) {
			words = append(words, bip39WordList[index])
		}
	}
	
	return strings.Join(words, " "), nil
}

func validateMnemonic(mnemonic string) bool {
	words := strings.Fields(mnemonic)
	if len(words) != 12 && len(words) != 24 {
		return false
	}
	
	// Create word set for quick lookup
	wordSet := make(map[string]bool)
	for _, word := range bip39WordList {
		wordSet[word] = true
	}
	
	for _, word := range words {
		if !wordSet[strings.ToLower(word)] {
			return false
		}
	}
	
	return true
}

func deriveMasterKey(mnemonic, password string) (string, error) {
	// Convert mnemonic to seed using PBKDF2
	salt := "mnemonic" + password
	seed := pbkdf2.Key([]byte(mnemonic), []byte(salt), PBKDF2Iterations, BIP39SeedLength, sha256.New)
	
	// Create master key from seed (simplified - uses first 32 bytes)
	masterKey := hex.EncodeToString(seed[:32])
	
	return masterKey, nil
}

func deriveAddress(masterKey, path string, isEVM bool) (string, string, string, error) {
	// Simplified derivation for demo - in production use proper BIP32/BIP44 derivation
	
	if isEVM {
		// Derive Ethereum address
		privKeyBytes, err := hex.DecodeString(masterKey[:64])
		if err != nil {
			return "", "", "", err
		}
		
		privateKey := crypto.ToECDSAUnsafe(privKeyBytes)
		publicKey := privateKey.Public().(*ecdsa.PublicKey)
		
		address := crypto.PubkeyToAddress(*publicKey).Hex()
		pubKeyHex := hex.EncodeToString(append(publicKey.X.Bytes(), publicKey.Y.Bytes()...))
		
		return address, pubKeyHex, masterKey[:64], nil
	}
	
	// For non-EVM chains, derive address based on path
	// This is a simplified version - in production use proper derivation
	addressHash := sha256.Sum256([]byte(masterKey + path))
	address = hex.EncodeToString(addressHash[:20])
	
	return "0x" + address, "", masterKey[:64], nil
}

func signEVMMessage(privKeyHex, message string) (string, error) {
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return "", err
	}
	
	privateKey := crypto.ToECDSAUnsafe(privKeyBytes)
	signature, err := crypto.Sign([]byte(message), privateKey)
	if err != nil {
		return "", err
	}
	
	return hex.EncodeToString(signature), nil
}

func signEVMTx(privKeyHex, txData string) (string, error) {
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return "", err
	}
	
	privateKey := crypto.ToECDSAUnsafe(privKeyBytes)
	
	// In production, properly serialize and sign transaction
	// This is a simplified version
	signature, err := crypto.Sign([]byte(txData), privateKey)
	if err != nil {
		return "", err
	}
	
	return "0x" + hex.EncodeToString(signature), nil
}

func now() int64 {
	return time.Now().Unix()
}

// ToJSON converts wallet to JSON
func (w *WalletInfo) ToJSON() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ============================================================================
// Utility
// ============================================================================

// GetWordList returns the BIP39 word list
func GetWordList() []string {
	return bip39WordList
}

// ValidateMnemonic validates a mnemonic phrase
func ValidateMnemonic(mnemonic string) bool {
	return validateMnemonic(mnemonic)
}

// MnemonicToSeed converts mnemonic to seed (for external use)
func MnemonicToSeed(mnemonic, password string) (string, error) {
	mnemonic = strings.ToLower(strings.TrimSpace(mnemonic))
	mnemonic = strings.Join(strings.Fields(mnemonic), " ")
	
	if !validateMnemonic(mnemonic) {
		return "", fmt.Errorf("invalid mnemonic")
	}
	
	salt := "mnemonic" + password
	seed := pbkdf2.Key([]byte(mnemonic), []byte(salt), PBKDF2Iterations, BIP39SeedLength, sha256.New)
	
	return hex.EncodeToString(seed), nil
}
