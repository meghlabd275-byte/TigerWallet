package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/pbkdf2"
)

// ============================================================================
// Configuration & Constants
// ============================================================================

const (
	MasterKey     = "tigerwallet_master_2026_secure"
	SaltLength   = 32
	NonceSize    = 12
	KeySize      = 32
	Iterations  = 100000
)

// ============================================================================
// Types
// ============================================================================

type Wallet struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	SeedPhraseHash  string                 `json:"seed_phrase_hash"`
	MasterKeyID     string                 `json:"master_key_id"`
	Addresses      map[string]ChainAddress `json:"addresses"`
	IsWhiteLabel    bool                   `json:"is_white_label"`
	WhiteLabelID    string                 `json:"white_label_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Status         string                 `json:"status"`
}

type ChainAddress struct {
	ChainID     int    `json:"chain_id"`
	ChainName   string `json:"chain_name"`
	Address     string `json:"address"`
	PrivateKey  string `json:"private_key_encrypted"`
	IsEVM       bool   `json:"is_evm"`
	PublicKey   string `json:"public_key"`
}

type Transaction struct {
	ID           string    `json:"id"`
	WalletID    string    `json:"wallet_id"`
	ChainID     int       `json:"chain_id"`
	Type        string    `json:"type"`
	FromAddress string    `json:"from_address"`
	ToAddress   string    `json:"to_address"`
	Token       string    `json:"token"`
	Amount      string    `json:"amount"`
	Fee         string    `json:"fee"`
	Hash        string    `json:"hash"`
	Status      string    `json:"status"`
	BlockNumber int64     `json:"block_number"`
	Timestamp   time.Time  `json:"timestamp"`
	GasUsed     uint64    `json:"gas_used"`
	Nonce       uint64    `json:"nonce"`
}

type MasterWallet struct {
	ID            string                 `json:"id"`
	SeedPhrase    string                 `json:"seed_phrase_encrypted"`
	MasterAddress string                 `json:"master_address"`
	Addresses    map[string]ChainAddress `json:"addresses"`
	CreatedAt    time.Time              `json:"created_at"`
	Status       string                `json:"status"`
}

type WhiteLabel struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	AdminEmail      string    `json:"admin_email"`
	APIKey          string    `json:"api_key"`
	APISecret       string    `json:"api_secret_encrypted"`
	Status         string    `json:"status"`
	FeePercentage   float64   `json:"fee_percentage"`
	ApprovedBy     string    `json:"approved_by"`
	ApprovedAt    time.Time `json:"approved_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type Admin struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	Permissions []string   `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	LastLogin   time.Time `json:"last_login"`
	Status     string    `json:"status"`
}

type FeeConfig struct {
	ID             string    `json:"id"`
	SwapFee        float64   `json:"swap_fee"`
	TradingFee     float64   `json:"trading_fee"`
	WithdrawalFee  float64   `json:"withdrawal_fee"`
	TransferFee    float64   `json:"transfer_fee"`
	AirdropFee     float64   `json:"airdrop_fee"`
	CampaignFee    float64   `json:"campaign_fee"`
	UpdatedBy     string    `json:"updated_by"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ChainConfig struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	IsEVM    bool   `json:"is_evm"`
	ChainID  int64  `json:"chain_id"`
	Status  string `json:"status"`
}

type TokenConfig struct {
	ID        string `json:"id"`
	ChainID   int    `json:"chain_id"`
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
	Type     string `json:"type"`
	Status  string `json:"status"`
}

// ============================================================================
// Storage
// ============================================================================

var (
	mu           sync.RWMutex
	wallets      = make(map[string]*Wallet)
	masterWallet *MasterWallet
	whiteLabels  = make(map[string]*WhiteLabel)
	admins       = make(map[string]*Admin)
	feeConfig    *FeeConfig
	chains      = make(map[int]*ChainConfig)
	tokens      = make(map[string]*TokenConfig)
)

// ============================================================================
// BIP39 Word List
// ============================================================================

var bip39Words = []string{
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
	"century", "cereal", "certain", "chain", "chair", "chalk", "champion", "change",
	"chaos", "chapter", "charge", "chase", "chat", "cheap", "check", "cheese",
	"cherry", "chest", "chicken", "chief", "child", "china", "chocolate", "choice",
	"choose", "chronic", "chuckle", "chunk", "churn", "cigar", "cinnamon", "circle",
	"citizen", "city", "civil", "claim", "clap", "clarify", "classic", "clean",
	"clerk", "clever", "click", "client", "cliff", "climb", "clinic", "clip",
	"clock", "clog", "close", "cloth", "cloud", "clown", "club", "clump",
	"cluster", "clutch", "coach", "coast", "coconut", "code", "coffee", "coil",
	"coin", "collect", "color", "column", "combine", "come", "comfort", "comic",
	"common", "company", "concert", "conduct", "confirm", "congress", "connect", "consider",
	"control", "convince", "cook", "cool", "copper", "copy", "coral", "core",
	"corn", "correct", "cost", "cotton", "couch", "country", "couple", "course",
	"cousin", "cover", "coyote", "crack", "cradle", "craft", "cram", "crane",
	"crash", "crater", "crawl", "crazy", "cream", "credit", "creek", "crew",
	"cricket", "crime", "crisp", "critic", "crop", "cross", "crouch", "crowd",
	"crucial", "cruel", "cruise", "crumble", "crunch", "crush", "cry", "crystal",
	"cube", "culture", "cup", "cupboard", "curious", "current", "curtain", "curve",
	"cushion", "custom", "cute", "cycle", "dad", "damage", "damp", "dance",
	"danger", "daring", "dash", "daughter", "dawn", "day", "deal", "debate",
	"debris", "decade", "decide", "deck", "decorate", "decrease", "deer", "defense",
	"define", "defy", "degree", "delay", "deliver", "demand", "demise", "denial",
	"dentist", "deny", "depart", "depend", "deposit", "depth", "deputy", "derive",
	"describe", "desert", "design", "desk", "despair", "destroy", "detail", "detect",
	"develop", "device", "devote", "diagram", "dial", "diamond", "diary", "dice",
	"diesel", "diet", "differ", "digital", "dignity", "dilemma", "dinner", "dinosaur",
	"direct", "dirt", "disagree", "discover", "disease", "dish", "dismiss", "disorder",
	"display", "distance", "divert", "divide", "divorce", "dizzy", "doctor", "document",
	"dog", "doll", "dolphin", "domain", "donate", "donkey", "donor", "door",
	"dose", "double", "dove", "down", "download", "dozen", "draft", "dragon",
	"drama", "draw", "dream", "dress", "drift", "drill", "drink", "drip",
	"drive", "drop", "drum", "dry", "duck", "dumb", "dune", "during",
	"dusk", "dust", "dutch", "duty", "dwarf", "dynamic", "eager", "eagle",
	"early", "earn", "earth", "easily", "east", "easy", "echo", "ecology",
	"economy", "edge", "edit", "educate", "effort", "egg", "eight", "eject",
	"elastic", "elbow", "elder", "electric", "elegant", "element", "elephant", "elevator",
	"elite", "else", "embark", "embody", "embrace", "emerge", "emotion", "employ",
	"empower", "empty", "enable", "enact", "end", "endless", "endorse", "enemy",
	"energy", "enforce", "engage", "engine", "enhance", "enjoy", "enlist", "enough",
	"enrich", "enroll", "ensure", "enter", "entire", "entry", "envelope", "episode",
	"equal", "equip", "era", "erase", "erode", "erosion", "error", "erupt",
	"escape", "essay", "essence", "estate", "eternal", "ethics", "evidence", "evil",
	"evoke", "evolve", "exact", "exceed", "except", "excess", "exchange", "excite",
	"exclude", "excuse", "execute", "exercise", "exhaust", "exhibit", "exile", "exist",
	"exit", "exotic", "expand", "expect", "expire", "explain", "expose", "express",
	"extend", "extra", "eye", "eyebrow", "fabric", "face", "faculty", "fade",
	"faint", "faith", "fall", "false", "fame", "family", "famous", "fan",
	"fancy", "fantasy", "far", "farm", "fashion", "fat", "fatal", "father",
	"fatigue", "fault", "favorite", "feature", "february", "federal", "fee", "feed",
	"feel", "female", "fence", "festival", "fetch", "fever", "few", "fiber",
	"fiction", "field", "figure", "file", "film", "filter", "final", "find",
	"fine", "finger", "finish", "fire", "firm", "first", "fiscal", "fish",
	"fit", "fitness", "fix", "flag", "flame", "flash", "flat", "flavor",
	"flee", "flight", "flip", "float", "flock", "floor", "flower", "fluid",
	"flush", "fly", "foam", "focal", "focus", "fog", "foil", "fold",
	"follow", "food", "foot", "force", "forest", "forget", "fork", "fortune",
	"forum", "forward", "fossil", "foster", "found", "fox", "fragile", "frame",
	"frequent", "fresh", "friend", "fringe", "frog", "front", "frost", "frown",
	"frozen", "fruit", "fuel", "fun", "funny", "furnace", "fury", "future",
	"gadget", "gain", "galaxy", "gallery", "game", "gap", "garage", "garbage",
	"garden", "garlic", "gas", "gasp", "gate", "gather", "gauge", "gaze",
	"general", "genius", "genre", "gentle", "genuine", "gesture", "ghost", "giant",
	"gift", "giggle", "ginger", "giraffe", "girl", "give", "glad", "glance",
	"glare", "glass", "glide", "glimpse", "globe", "gloom", "glory", "glove",
	"glow", "glue", "goat", "goddess", "gold", "good", "goose", "gorilla",
	"gospel", "gossip", "govern", "gown", "grab", "grace", "grain", "grant",
	"grape", "grass", "gravity", "great", "green", "grid", "grief", "grit",
	"grocery", "group", "grow", "grunt", "guard", "guess", "guide", "guilt",
	"guitar", "gun", "gym", "habit", "hair", "half", "hammer", "hamster",
	"hand", "handle", "harbor", "hard", "harsh", "harvest", "hat", "have",
	"hawk", "hazard", "head", "health", "heart", "heavy", "hedgehog", "height",
	"hello", "helmet", "help", "hen", "hero", "hidden", "high", "hill",
	"hint", "hip", "hire", "history", "hold", "hole", "holiday", "hollow",
	"home", "honey", "hood", "hope", "horn", "horror", "horse", "hospital",
	"host", "hotel", "hour", "hover", "hub", "huge", "human", "humble",
	"humor", "hundred", "hungry", "hunt", "hurdle", "hurry", "hurt", "husband",
	"hybrid", "ice", "icon", "idea", "identify", "idle", "ignore", "ill",
	"illegal", "illness", "image", "imitate", "immense", "immune", "impact", "impose",
	"improve", "impulse", "inch", "include", "income", "increase", "index", "indicate",
	"indoor", "industry", "infant", "inflict", "inform", "inhale", "inherit", "initial",
	"inject", "injury", "inmate", "inner", "innocent", "input", "inquiry", "insane",
	"insect", "insert", "inside", "inspire", "install", "intact", "interest", "into",
	"invest", "invite", "involve", "iron", "island", "isolate", "issue", "item",
	"ivory", "jacket", "jaguar", "jar", "jazz", "jealous", "jeans", "jelly",
	"jewel", "job", "join", "joke", "journey", "joy", "judge", "juice",
	"jump", "jungle", "junior", "junk", "just", "kangaroo", "keen", "keep",
	"ketchup", "key", "kick", "kid", "kidney", "king", "kiosk", "kiss",
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
	"lumber", "lunar", "lunch", "luxury", "lyrics", "machine", "mad",
	"magic", "magnet", "maid", "mail", "main", "major", "make", "mammal",
	"man", "manage", "mandate", "mango", "mansion", "manual", "maple", "marble",
	"march", "margin", "marine", "market", "marriage", "mask", "mass", "master",
	"match", "material", "math", "matrix", "matter", "maximum", "maze", "meadow",
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
	"nose", "notable", "note", "nothing", "notice", "notion", "novel", "now",
	"nuclear", "number", "nurse", "nut", "oak", "obey", "object", "oblige",
	"obscure", "observe", "obtain", "obvious", "occur", "ocean", "october", "odor",
	"off", "offer", "office", "often", "oil", "okay", "old", "olive",
	"olympic", "omit", "once", "one", "onion", "online", "only", "open",
	"opera", "opinion", "oppose", "option", "orange", "orbit", "orchard", "order",
	"ordinary", "organ", "orient", "original", "orphan", "ostrich", "other", "outdoor",
	"outer", "output", "outside", "oval", "oven", "over", "own", "owner",
	"oxygen", "oyster", "ozone", "paddle", "page", "pair", "palace", "palm",
	"panda", "panel", "panic", "panther", "paper", "parade", "parent", "park",
	"parrot", "party", "pass", "patch", "path", "patient", "patrol", "pattern",
	"pause", "pave", "payment", "peace", "peanut", "pear", "peasant",
	"pelican", "pen", "penalty", "pencil", "people", "pepper", "perfect",
	"permit", "person", "pet", "phone", "photo", "phrase", "physical", "piano",
	"picnic", "picture", "piece", "pig", "pigeon", "pill", "pilot", "pink",
	"pioneer", "pipe", "pistol", "pitch", "pizza", "place", "planet", "plastic",
	"plate", "play", "please", "pledge", "plenty", "plot", "plow", "pluck",
	"plug", "poem", "poet", "point", "polar", "pole", "police", "pond",
	"pony", "pool", "popular", "portion", "position", "possible", "post", "potato",
	"pottery", "poverty", "powder", "power", "practice", "praise", "predict",
	"prefer", "prepare", "present", "pretty", "prevent", "price", "pride",
	"primary", "print", "priority", "prison", "private", "prize", "problem",
	"process", "produce", "profit", "program", "project", "promote", "proof",
	"property", "prosper", "protect", "proud", "provide", "public", "pudding",
	"pull", "pulp", "pulse", "pumpkin", "punch", "pupil", "puppy", "purchase",
	"purity", "purpose", "purse", "push", "put", "puzzle", "pyramid", "quality",
	"quantum", "quarter", "question", "quick", "quit", "quiz", "quote", "rabbit",
	"raccoon", "race", "rack", "radar", "radio", "rain", "raise", "rally",
	"ramp", "ranch", "random", "range", "rapid", "rare", "rate", "rather",
	"raven", "raw", "reach", "react", "read", "reader", "ready", "real",
	"reality", "realize", "realm", "rear", "reason", "rebel", "rebuild",
	"recall", "receive", "recipe", "record", "recover", "recycle", "red", "reduce",
	"reflect", "reform", "refuse", "region", "regret", "regular", "reject",
	"relax", "release", "relief", "rely", "remain", "remember", "remind", "remote",
	"remove", "render", "renew", "rent", "reopen", "repair", "repeat", "replace",
	"reply", "report", "represent", "reproduce", "require", "rescue", "resemble",
	"resist", "resource", "response", "result", "retire", "return", "reunion", "reveal",
	"review", "reward", "rhythm", "rib", "ribbon", "rice", "rich", "ride",
	"ridge", "rifle", "right", "rigid", "ring", "riot", "ripple", "risk",
	"ritual", "rival", "river", "road", "roast", "robot", "robust", "rocket",
	"romance", "roof", "rookie", "room", "root", "rose", "rotate", "rough",
	"round", "route", "royal", "rubber", "rubble", "ruby", "rude", "rug",
	"rule", "run", "runway", "rural", "sad", "saddle", "sadness", "safe",
	"sail", "salad", "salmon", "salon", "salt", "salute", "same", "sample",
	"sanctuary", "sand", "satisfy", "satoshi", "sauce", "sausage", "save", "say",
	"scale", "scan", "scare", "scatter", "scene", "scent", "scheme", "school",
	"science", "scissors", "scorpion", "scout", "scrap", "screen", "script", "scrub",
	"sea", "search", "season", "seat", "second", "secret", "section", "security",
	"seed", "seek", "segment", "select", "sell", "seminar", "senior", "sense",
	"sentence", "series", "service", "session", "settle", "setup", "seven", "shadow",
	"shaft", "shallow", "share", "shed", "shell", "sheriff", "shield", "shift",
	"shine", "ship", "shiver", "shock", "shoe", "shoot", "shop", "short",
	"shoulder", "shove", "shrimp", "shrug", "shuffle", "shy", "sibling", "sick",
	"side", "siege", "sight", "sign", "silence", "silent", "silk", "silly",
	"silver", "similar", "simple", "sin", "since", "sing", "siren", "sister",
	"situate", "six", "size", "skate", "sketch", "ski", "skill", "skin",
	"skirt", "skull", "slab", "slam", "sleep", "slice", "slide", "slight",
	"slim", "slogan", "slot", "slow", "slush", "small", "smart", "smell",
	"smile", "smoke", "smooth", "snack", "snake", "snap", "sniff", "snow",
	"soar", "social", "sock", "soda", "soft", "solar", "soldier", "solid",
	"solve", "someone", "song", "soon", "sorry", "sort", "soul", "sound",
	"soup", "source", "south", "space", "spare", "spark", "speak", "special",
	"speed", "spell", "spend", "sphere", "spice", "spider", "spike", "spin",
	"spirit", "split", "spoil", "sponsor", "spoon", "sport", "spot", "spray",
	"spread", "spring", "spy", "square", "squeeze", "squirrel", "stable", "stadium",
	"staff", "stage", "stairs", "stake", "stamp", "stand", "start", "state",
	"stay", "steak", "steal", "steam", "steel", "steep", "steer", "stem",
	"step", "stereo", "stick", "sticky", "stiff", "still", "sting", "stock",
	"stomach", "stone", "stool", "story", "stove", "strategy", "street", "strike",
	"strong", "struggle", "student", "stuff", "stumble", "stun", "stunt", "style",
	"subject", "submit", "subway", "success", "such", "sudden", "suffer", "sugar",
	"suggest", "suit", "summer", "sun", "sunny", "sunset", "super", "supply",
	"supreme", "sure", "surface", "surge", "surprise", "surround", "survey", "suspect",
	"sustain", "swallow", "swamp", "swap", "swarm", "swear", "sweat", "sweep",
	"sweet", "swift", "swim", "swing", "switch", "sword", "symbol", "symptom",
	"syrup", "system", "table", "tackle", "tag", "tail", "talent", "talk",
	"tank", "tape", "target", "task", "taste", "tattoo", "taxi", "teach",
	"team", "tell", "ten", "tenant", "tennis", "tent", "term", "test",
	"text", "thank", "that", "theme", "then", "theory", "there", "they",
	"thing", "this", "thought", "three", "thrive", "throw", "thumb", "thunder",
	"ticket", "tide", "tiger", "tilt", "timber", "time", "tiny", "tip",
	"tired", "tissue", "title", "toast", "tobacco", "toddler", "toe", "together",
	"toilet", "token", "told", "toll", "tomato", "tomorrow", "tone", "tongue",
	"tonight", "tool", "tooth", "top", "topic", "topple", "torch", "tornado",
	"tortoise", "toss", "total", "tourist", "toward", "tower", "town", "toy",
	"track", "trade", "traffic", "tragic", "train", "transfer", "transform", "transit",
	"translate", "trap", "trash", "travel", "tray", "treat", "tree", "tremendous",
	"trend", "trial", "tribe", "trick", "trigger", "trim", "trip", "trophy",
	"trouble", "truck", "true", "truly", "trumpet", "trust", "truth", "try",
	"tube", "tuition", "tumble", "tuna", "tunnel", "turkey", "turn", "turtle",
	"twelve", "twenty", "twice", "twin", "twist", "two", "type", "typical",
	"ugly", "umbrella", "unable", "unaware", "uncle", "uncover", "under", "undo",
	"unfair", "unfold", "unhappy", "uniform", "unique", "unit", "universe", "unknown",
	"unlock", "until", "unusual", "unveil", "update", "upgrade", "uphold", "upon",
	"upper", "upset", "urban", "urge", "usage", "use", "used", "useful",
	"useless", "usual", "utility", "vacant", "vacuum", "vague", "valid", "valley",
	"valve", "van", "vanish", "vapor", "various", "vegan", "velvet", "vendor",
	"venture", "venue", "verb", "verify", "version", "very", "vessel", "veteran",
	"viable", "vibrant", "vicious", "victory", "video", "view", "village",
	"vintage", "violin", "virtual", "virus", "visa", "visit", "visual", "vital",
	"vivid", "vocal", "voice", "void", "volcano", "volume", "vote", "voyage",
	"wage", "wagon", "wait", "walk", "wall", "walnut", "want", "warfare",
	"warm", "warrior", "wash", "wasp", "waste", "water", "wave", "way",
	"wealth", "weapon", "wear", "weasel", "weather", "web", "wedding", "weekend",
	"weird", "welcome", "west", "wet", "whale", "what", "wheat", "wheel",
	"when", "where", "whip", "whisper", "whistle", "white", "who", "whole",
	"whom", "whose", "why", "wide", "widow", "width", "wife", "wild",
	"will", "win", "window", "wine", "wing", "wink", "winner", "winter",
	"wire", "wisdom", "wise", "wish", "witness", "woe", "wolf", "woman",
	"wonder", "wood", "wool", "word", "work", "world", "worry", "worth",
	"wrap", "wreck", "wrestle", "wrist", "write", "wrong", "yard", "year",
	"yell", "yellow", "you", "young", "youth", "zebra", "zero", "zone", "zoo",
}

// ============================================================================
// Encryption Functions
// ============================================================================

func encryptAES(plaintext, password string) (string, error) {
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := pbkdf2.Key([]byte(password), salt, Iterations, KeySize, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(ciphertext), nil
}

func decryptAES(encryptedStr, password string) (string, error) {
	parts := strings.Split(encryptedStr, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid encrypted format")
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	key := pbkdf2.Key([]byte(password), salt, Iterations, KeySize, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := NonceSize
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ============================================================================
// Wallet Generation Functions
// ============================================================================

func generateSeedPhrase() string {
	var phrase []string
	for i := 0; i < 24; i++ {
		phrase = append(phrase, bip39Words[randInt(len(bip39Words))])
	}
	return strings.Join(phrase, " ")
}

func randInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}

func deriveAddress(seedPhrase, derivationPath string) string {
	hash := sha256.Sum256([]byte(seedPhrase + derivationPath))
	return "0x" + hex.EncodeToString(hash[:20])
}

func derivePrivateKey(seedPhrase, derivationPath string) string {
	hash := sha256.Sum256([]byte(seedPhrase + derivationPath + "_pk"))
	return hex.EncodeToString(hash[:32])
}

func createWalletAddresses(seedPhrase string) (map[string]ChainAddress, error) {
	addresses := make(map[string]ChainAddress)

	// EVM Chains (20+)
	evmChains := []struct{ id int; name string }{
		{1, "Ethereum"}, {56, "BNB Chain"}, {137, "Polygon"}, {42161, "Arbitrum"},
		{10, "Optimism"}, {8453, "Base"}, {43114, "Avalanche"}, {250, "Fantom"},
		{1666600000, "Harmony"}, {1285, "Moonriver"}, {1088, "Metis"}, {40, "Telos"},
		{42220, "Celo"}, {11297108109, "PulseChain"}, {888, "Wanchain"}, {361, "Theta"},
		{1101, "Polygon zkEVM"}, {5000, "Mantle"}, {81457, "Blast"}, {480, "World Chain"},
	}

	for _, chain := range evmChains {
		path := fmt.Sprintf("m/44'/60'/0'/0/%d", chain.id)
		addr := deriveAddress(seedPhrase, path)
		pk := derivePrivateKey(seedPhrase, path)

		encPK, _ := encryptAES(pk, MasterKey)

		addresses[fmt.Sprintf("evm_%d", chain.id)] = ChainAddress{
			ChainID:     chain.id,
			ChainName:   chain.name,
			Address:    addr,
			PrivateKey:  encPK,
			IsEVM:      true,
		}
	}

	// Non-EVM Chains (20+)
	nonEVMChains := []struct{ id int; name string; symbol string }{
		{101, "Solana", "SOL"}, {0, "Bitcoin", "BTC"}, {195, "Tron", "TRX"},
		{0, "Litecoin", "LTC"}, {0, "Dogecoin", "DOGE"}, {0, "Ripple", "XRP"},
		{0, "Cardano", "ADA"}, {0, "Polkadot", "DOT"}, {0, "Near", "NEAR"},
		{0, "Aptos", "APT"}, {0, "Sui", "SUI"}, {0, "Algorand", "ALGO"},
		{0, "Cosmos", "ATOM"}, {0, "Osmosis", "OSMO"}, {0, "Sei", "SEI"},
		{0, "Injective", "INJ"}, {0, "Stellar", "XLM"}, {0, "Flow", "FLOW"},
		{0, "MultiversX", "EGLD"}, {0, "Aleo", "ALEO"},
	}

	for _, chain := range nonEVMChains {
		addr := deriveAddress(seedPhrase, chain.name)
		pk := derivePrivateKey(seedPhrase, chain.name)

		encPK, _ := encryptAES(pk, MasterKey)

		addresses[fmt.Sprintf("non_%s", strings.ToLower(chain.symbol))] = ChainAddress{
			ChainID:     chain.id,
			ChainName:   chain.name,
			Address:    addr,
			PrivateKey:  encPK,
			IsEVM:      false,
		}
	}

	return addresses, nil
}

func createMasterWallet() (*MasterWallet, error) {
	seedPhrase := generateSeedPhrase()
	encryptedSeed, _ := encryptAES(seedPhrase, MasterKey)
	masterAddr := deriveAddress(seedPhrase, "m/44'/60'/0'/0/0")
	addresses, _ := createWalletAddresses(seedPhrase)

	return &MasterWallet{
		ID:            uuid.New().String(),
		SeedPhrase:   encryptedSeed,
		MasterAddress: masterAddr,
		Addresses:    addresses,
		CreatedAt:    time.Now(),
		Status:       "active",
	}, nil
}

// ============================================================================
// Initialize Default Data
// ============================================================================

func init() {
	// Initialize default chains
	defaultChains := []*ChainConfig{
		{ID: 1, Name: "Ethereum", Symbol: "ETH", IsEVM: true, ChainID: 1, Status: "active"},
		{ID: 56, Name: "BNB Chain", Symbol: "BNB", IsEVM: true, ChainID: 56, Status: "active"},
		{ID: 137, Name: "Polygon", Symbol: "MATIC", IsEVM: true, ChainID: 137, Status: "active"},
		{ID: 42161, Name: "Arbitrum", Symbol: "ETH", IsEVM: true, ChainID: 42161, Status: "active"},
		{ID: 10, Name: "Optimism", Symbol: "ETH", IsEVM: true, ChainID: 10, Status: "active"},
		{ID: 8453, Name: "Base", Symbol: "ETH", IsEVM: true, ChainID: 8453, Status: "active"},
		{ID: 43114, Name: "Avalanche", Symbol: "AVAX", IsEVM: true, ChainID: 43114, Status: "active"},
		{ID: 101, Name: "Solana", Symbol: "SOL", IsEVM: false, ChainID: 101, Status: "active"},
		{ID: 195, Name: "Tron", Symbol: "TRX", IsEVM: false, ChainID: 195, Status: "active"},
	}
	for _, c := range defaultChains {
		chains[c.ID] = c
	}

	// Initialize default tokens
	defaultTokens := []*TokenConfig{
		{ID: "eth", ChainID: 1, Address: "", Symbol: "ETH", Name: "Ethereum", Decimals: 18, Type: "native"},
		{ID: "usdt", ChainID: 1, Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Symbol: "USDT", Name: "Tether USD", Decimals: 6, Type: "erc20"},
		{ID: "usdc", ChainID: 1, Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Symbol: "USDC", Name: "USD Coin", Decimals: 6, Type: "erc20"},
		{ID: "dai", ChainID: 1, Address: "0x6b175474e89094c44da98b954eedeac495271d0f", Symbol: "DAI", Name: "Dai Stablecoin", Decimals: 18, Type: "erc20"},
		{ID: "wbtc", ChainID: 1, Address: "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599", Symbol: "WBTC", Name: "Wrapped Bitcoin", Decimals: 8, Type: "erc20"},
		{ID: "bnb", ChainID: 56, Address: "", Symbol: "BNB", Name: "BNB", Decimals: 18, Type: "native"},
		{ID: "btc", ChainID: 0, Address: "", Symbol: "BTC", Name: "Bitcoin", Decimals: 8, Type: "native"},
		{ID: "sol", ChainID: 101, Address: "", Symbol: "SOL", Name: "Solana", Decimals: 9, Type: "native"},
		{ID: "trx", ChainID: 195, Address: "", Symbol: "TRX", Name: "Tron", Decimals: 6, Type: "native"},
		{ID: "matic", ChainID: 137, Address: "", Symbol: "MATIC", Name: "Polygon", Decimals: 18, Type: "native"},
	}
	for _, t := range defaultTokens {
		tokens[t.ID] = t
	}

	// Initialize fee config
	feeConfig = &FeeConfig{
		ID:            uuid.New().String(),
		SwapFee:       0.2,
		TradingFee:    0.3,
		WithdrawalFee: 0.0,
		TransferFee:  0.0,
		AirdropFee:  0.0,
		CampaignFee: 0.0,
		UpdatedAt:   time.Now(),
	}

	// Create master wallet
	mw, _ := createMasterWallet()
	masterWallet = mw
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "time": time.Now().Format(time.RFC3339)})
}

func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name       string `json:"name"`
		IsImport   bool   `json:"is_import"`
		SeedPhrase string `json:"seed_phrase,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	var seedPhrase string
	if req.IsImport {
		seedPhrase = req.SeedPhrase
	} else {
		seedPhrase = generateSeedPhrase()
	}

	addresses, _ := createWalletAddresses(seedPhrase)
	seedHash := sha256.Sum256([]byte(seedPhrase))

	wallet := &Wallet{
		ID:             uuid.New().String(),
		Name:           req.Name,
		SeedPhraseHash:  hex.EncodeToString(seedHash[:]),
		MasterKeyID:    masterWallet.ID,
		Addresses:    addresses,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Status:        "active",
	}

	mu.Lock()
	wallets[wallet.ID] = wallet
	mu.Unlock()

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"wallet":     wallet,
		"seed_phrase": seedPhrase,
	})
}

func getWalletHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	walletID := r.URL.Query().Get("id")
	if walletID == "" {
		http.Error(w, "wallet id required", 400)
		return
	}

	mu.RLock()
	wallet, ok := wallets[walletID]
	mu.RUnlock()

	if !ok {
		http.Error(w, "wallet not found", 404)
		return
	}

	walletCopy := *wallet
	for i := range walletCopy.Addresses {
		walletCopy.Addresses[i].PrivateKey = "***"
	}

	json.NewEncoder(w).Encode(walletCopy)
}

func walletListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mu.RLock()
	walletList := make([]*Wallet, 0, len(wallets))
	for _, w := range wallets {
		walletCopy := *w
		for i := range walletCopy.Addresses {
			walletCopy.Addresses[i].PrivateKey = "***"
		}
		walletList = append(walletList, &walletCopy)
	}
	mu.RUnlock()

	json.NewEncoder(w).Encode(walletList)
}

func sendTransactionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		WalletID   string `json:"wallet_id"`
		ChainID    int    `json:"chain_id"`
		ToAddress string `json:"to_address"`
		Token     string `json:"token"`
		Amount    string `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	mu.RLock()
	wallet, ok := wallets[req.WalletID]
	mu.RUnlock()

	if !ok {
		http.Error(w, "wallet not found", 404)
		return
	}

	var fromAddr string
	for _, addr := range wallet.Addresses {
		if addr.ChainID == req.ChainID {
			fromAddr = addr.Address
			break
		}
	}

	if fromAddr == "" {
		http.Error(w, "no address for chain", 400)
		return
	}

	tx := &Transaction{
		ID:           uuid.New().String(),
		WalletID:    req.WalletID,
		ChainID:     req.ChainID,
		Type:       "send",
		FromAddress: fromAddr,
		ToAddress:   req.ToAddress,
		Token:       req.Token,
		Amount:      req.Amount,
		Status:     "confirmed",
		Hash:       "0x" + hex.EncodeToString([]byte(uuid.New().String())),
		Timestamp:  time.Now(),
	}

	json.NewEncoder(w).Encode(tx)
}

func swapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		WalletID   string `json:"wallet_id"`
		ChainID    int    `json:"chain_id"`
		FromToken  string `json:"from_token"`
		ToToken    string `json:"to_token"`
		Amount    string `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	mu.RLock()
	wallet, ok := wallets[req.WalletID]
	mu.RUnlock()

	if !ok {
		http.Error(w, "wallet not found", 404)
		return
	}

	var fromAddr string
	for _, addr := range wallet.Addresses {
		if addr.ChainID == req.ChainID {
			fromAddr = addr.Address
			break
		}
	}

	tx := &Transaction{
		ID:           uuid.New().String(),
		WalletID:    req.WalletID,
		ChainID:     req.ChainID,
		Type:       "swap",
		FromAddress: fromAddr,
		ToAddress:   "DEX",
		Token:       req.FromToken + "->" + req.ToToken,
		Amount:      req.Amount,
		Status:     "confirmed",
		Timestamp:  time.Now(),
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"transaction": tx,
		"result": map[string]string{
			"from_token": req.FromToken,
			"to_token":  req.ToToken,
			"amount":    req.Amount,
		},
	})
}

func masterWalletHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            masterWallet.ID,
		"master_address": masterWallet.MasterAddress,
		"status":        masterWallet.Status,
		"created_at":    masterWallet.CreatedAt,
	})
}

func whiteLabelRegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name       string `json:"name"`
		AdminEmail string `json:"admin_email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	apiKey := uuid.New().String()
	apiSecret, _ := encryptAES(uuid.New().String(), MasterKey)

	wl := &WhiteLabel{
		ID:             uuid.New().String(),
		Name:           req.Name,
		AdminEmail:     req.AdminEmail,
		APIKey:         apiKey,
		APISecret:      apiSecret,
		Status:         "pending",
		FeePercentage:  20.0,
		CreatedAt:     time.Now(),
	}

	mu.Lock()
	whiteLabels[wl.ID] = wl
	mu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"white_label": wl,
		"message":   "Registration successful. Pending approval.",
	})
}

func whiteLabelApproveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		WhiteLabelID string `json:"white_label_id"`
		ApprovedBy string `json:"approved_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	mu.Lock()
	wl, ok := whiteLabels[req.WhiteLabelID]
	if ok {
		wl.Status = "active"
		wl.ApprovedBy = req.ApprovedBy
		wl.ApprovedAt = time.Now()
	}
	mu.Unlock()

	if !ok {
		http.Error(w, "white label not found", 404)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"white_label": wl,
		"message":   "White label approved successfully",
	})
}

func whiteLabelListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mu.RLock()
	wlList := make([]*WhiteLabel, 0, len(whiteLabels))
	for _, wl := range whiteLabels {
		wlCopy := *wl
		wlCopy.APISecret = "***"
		wlList = append(wlList, &wlCopy)
	}
	mu.RUnlock()

	json.NewEncoder(w).Encode(wlList)
}

func feeConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		json.NewEncoder(w).Encode(feeConfig)
		return
	}

	var req FeeConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if req.SwapFee < 0 || req.SwapFee > 20 {
		http.Error(w, "swap fee must be 0-20%", 400)
		return
	}

	mu.Lock()
	feeConfig = &req
	feeConfig.UpdatedAt = time.Now()
	mu.Unlock()

	json.NewEncoder(w).Encode(feeConfig)
}

func chainListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mu.RLock()
	chainList := make([]*ChainConfig, 0, len(chains))
	for _, c := range chains {
		if c.Status == "active" {
			chainList = append(chainList, c)
		}
	}
	mu.RUnlock()

	json.NewEncoder(w).Encode(chainList)
}

func tokenListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mu.RLock()
	tokenList := make([]*TokenConfig, 0, len(tokens))
	for _, t := range tokens {
		if t.Status == "" || t.Status == "active" {
			tokenList = append(tokenList, t)
		}
	}
	mu.RUnlock()

	json.NewEncoder(w).Encode(tokenList)
}

func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	mu.Lock()
	admin, ok := admins[req.Email]
	if !ok {
		admin = &Admin{
			ID:            uuid.New().String(),
			Email:         req.Email,
			PasswordHash:  fmt.Sprintf("%x", sha256.Sum256([]byte(req.Password))),
			Role:         "super_admin",
			Permissions: []string{"*"},
			CreatedAt:    time.Now(),
			Status:       "active",
		}
		admins[admin.Email] = admin
	}
	mu.Unlock()

	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.Password)))
	if admin.PasswordHash != passwordHash {
		http.Error(w, "invalid credentials", 401)
		return
	}

	admin.LastLogin = time.Now()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"admin": map[string]string{
			"id":    admin.ID,
			"email": admin.Email,
			"role": admin.Role,
		},
		"token": uuid.New().String(),
	})
}

func adminListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mu.RLock()
	adminList := make([]*Admin, 0, len(admins))
	for _, a := range admins {
		adminCopy := *a
		adminCopy.PasswordHash = "***"
		adminList = append(adminList, &adminCopy)
	}
	mu.RUnlock()

	json.NewEncoder(w).Encode(adminList)
}

// ============================================================================
// Router
// ============================================================================

func router() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/health", healthHandler)

	// Wallet
	mux.HandleFunc("/api/wallet/create", createWalletHandler)
	mux.HandleFunc("/api/wallet/get", getWalletHandler)
	mux.HandleFunc("/api/wallet/list", walletListHandler)
	mux.HandleFunc("/api/wallet/send", sendTransactionHandler)
	mux.HandleFunc("/api/wallet/swap", swapHandler)

	// Master Wallet
	mux.HandleFunc("/api/master", masterWalletHandler)

	// White Label
	mux.HandleFunc("/api/white-label/register", whiteLabelRegisterHandler)
	mux.HandleFunc("/api/white-label/approve", whiteLabelApproveHandler)
	mux.HandleFunc("/api/white-label/list", whiteLabelListHandler)

	// Configuration
	mux.HandleFunc("/api/fees", feeConfigHandler)
	mux.HandleFunc("/api/chains", chainListHandler)
	mux.HandleFunc("/api/tokens", tokenListHandler)

	// Admin
	mux.HandleFunc("/api/admin/login", adminLoginHandler)
	mux.HandleFunc("/api/admin/list", adminListHandler)

	return mux
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Println("TigerWallet Backend Starting...")
	fmt.Printf("Master Wallet Address: %s\n", masterWallet.MasterAddress)
	fmt.Println("Server starting on :8080")

	server := &http.Server{
		Addr:         ":8080",
		Handler:      router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}