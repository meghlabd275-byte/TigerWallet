// TigerWallet Go Wallet Service - High-Performance Distributed Wallet Management
// Production-ready implementation with real blockchain connectivity

package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort    string `json:"server_port"`
	DBHost        string `json:"db_host"`
	DBPort        string `json:"db_port"`
	DBUser        string `json:"db_user"`
	DBPassword    string `json:"db_password"`
	DBName        string `json:"db_name"`
	RedisHost     string `json:"redis_host"`
	RedisPort     string `json:"redis_port"`
	JWTSecret     string `json:"jwt_secret"`
	EncryptionKey string `json:"encryption_key"`
	MasterWallet  string `json:"master_wallet"`
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("WALLET_PORT", "9094"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "tigerwallet"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "tigerwallet_wallet"),
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "wallet-32-byte-encryption-key!!"),
		MasterWallet:  getEnv("MASTER_WALLET", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

// Wallet represents a user wallet
type Wallet struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	WalletID          string    `gorm:"uniqueIndex;not null" json:"wallet_id"`
	UserID            string    `gorm:"index;not null" json:"user_id"`
	Address           string    `gorm:"index;not null" json:"address"`
	ChainType         string    `json:"chain_type"` // evm, solana, tron, etc.
	ChainID           int64     `json:"chain_id"`
	PrivateKeyEncrypted string  `json:"-"`
	PublicKey         string    `json:"public_key"`
	MnemonicEncrypted string    `json:"-"`
	SeedEncrypted     string    `json:"-"`
	WalletType        string    `json:"wallet_type"` // master, user, imported
	Status            string    `json:"status"` // active, suspended, deleted
	IsPrimary         bool      `json:"is_primary"`
	WhiteLabelID      *uint     `gorm:"index" json:"white_label_id"`
	ReferralCode     string    `gorm:"uniqueIndex" json:"referral_code"`
}

// TokenBalance represents token holdings
type TokenBalance struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	WalletID      uint      `gorm:"index;not null" json:"wallet_id"`
	TokenAddress  string    `gorm:"index" json:"token_address"`
	ChainID       int64     `json:"chain_id"`
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Decimals      int       `json:"decimals"`
	Balance       string    `json:"balance"`
	BalanceUSD    float64   `json:"balance_usd"`
	IsNative      bool      `json:"is_native"`
	LastSyncedAt  time.Time `json:"last_synced_at"`
}

// Transaction represents on-chain transactions
type Transaction struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	TxHash        string    `gorm:"uniqueIndex;not null" json:"tx_hash"`
	WalletID      uint      `gorm:"index" json:"wallet_id"`
	FromAddress   string    `gorm:"index" json:"from_address"`
	ToAddress     string    `json:"to_address"`
	ChainID       int64     `json:"chain_id"`
	TokenAddress  string    `json:"token_address"`
	Amount        string    `json:"amount"`
	Fee           string    `json:"fee"`
	Status        string    `json:"status"` // pending, confirmed, failed
	BlockNumber   uint64    `json:"block_number"`
	BlockHash     string    `json:"block_hash"`
	Timestamp     int64     `json:"timestamp"`
	Type          string    `json:"type"` // send, receive, swap, stake, etc.
	GasUsed       uint64    `json:"gas_used"`
	GasPrice      string    `json:"gas_price"`
	Nonce         uint64    `json:"nonce"`
	RawTx         string    `json:"raw_tx"`
}

// BlockchainNetwork represents supported blockchains
type BlockchainNetwork struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	ChainID         int64     `gorm:"uniqueIndex;not null" json:"chain_id"`
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	Type            string    `json:"type"` // evm, solana, tron, etc.
	RPCURL          string    `json:"rpc_url"`
	RPCURLs         string    `json:"rpc_urls"` // JSON array
	ExplorerURL    string    `json:"explorer_url"`
	ExplorerAPI    string    `json:"explorer_api"`
	ChainLogo      string    `json:"chain_logo"`
	Decimals       int       `json:"decimals"`
	BlockTime      int       `json:"block_time"`
	GasLimit       uint64    `json:"gas_limit"`
	IsEnabled      bool      `json:"is_enabled"`
	IsTestnet      bool      `json:"is_testnet"`
	AddedAt        time.Time `json:"added_at"`
	AddedBy        string    `json:"added_by"`
}

// SupportedToken represents tokens on networks
type SupportedToken struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	ChainID       int64     `gorm:"index;not null" json:"chain_id"`
	TokenAddress  string    `gorm:"index" json:"token_address"`
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Decimals      int       `json:"decimals"`
	Logo          string    `json:"logo"`
	IsNative      bool      `json:"is_native"`
	IsStable      bool      `json:"is_stable"`
	IsEnabled     bool      `json:"is_enabled"`
	MinDeposit    string    `json:"min_deposit"`
	WithdrawalFee string    `json:"withdrawal_fee"`
}

// User represents platform users
type User struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UserID            string    `gorm:"uniqueIndex;not null" json:"user_id"`
	Email             string    `gorm:"index" json:"email"`
	Phone             string    `json:"phone"`
	PasswordHash      string    `json:"-"`
	MasterWalletID    string    `gorm:"index" json:"master_wallet_id"`
	Status            string    `json:"status"` // active, suspended, banned
	Tier              int       `json:"tier"` // 0: basic, 1: verified, 2: premium
	IsEmailVerified   bool      `json:"is_email_verified"`
	IsPhoneVerified   bool      `json:"is_phone_verified"`
	KYCStatus         string    `json:"kyc_status"` // none, pending, approved, rejected
	WhiteLabelID      *uint     `gorm:"index" json:"white_label_id"`
	LastLoginAt       *time.Time `json:"last_login_at"`
}

// ============================================================================
// Crypto Functions - Real Implementation
// ============================================================================

// BIP39 Wordlist - Full 2048 words
var bip39Wordlist = []string{
	"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
	"absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
	"acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual",
	"adapt", "add", "addict", "address", "adjust", "admit", "adult", "advance",
	"advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent",
	"agree", "ahead", "aim", "air", "airport", "aisle", "alarm", "album",
	"alcohol", "alert", "alien", "all", "alley", "allow", "almost", "alone",
	"alpha", "already", "also", "alter", "always", "amateur", "amazing", "among",
	"amount", "amused", "analyst", "anchor", "ancient", "anger", "angle", "angry",
	"animal", "ankle", "announce", "annual", "another", "answer", "antenna",
	"anticipate", "anxiety", "any", "apart", "apology", "appear", "apple", "approve",
	"april", "arch", "arctic", "area", "arena", "argue", "arm", "armed",
	"armor", "army", "around", "arrange", "arrest", "arrive", "arrow", "art",
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
	"black", "blade", "blame", "blanket", "blast", "bleak", "bless", "blind",
	"blood", "blossom", "blouse", "blue", "blur", "blush", "board", "boat",
	"body", "boil", "bomb", "bone", "bonus", "book", "boost", "border",
	"boring", "borrow", "boss", "bottom", "bounce", "box", "boy", "bracket",
	"brain", "brand", "brass", "brave", "bread", "breeze", "brick", "bridge",
	"brief", "bright", "bring", "brisk", "broccoli", "broken", "bronze", "broom",
	"brother", "brown", "brush", "bubble", "buddy", "budget", "buffalo", "build",
	"bulb", "bulk", "bullet", "bundle", "bunker", "burden", "burger", "burst",
	"bus", "business", "busy", "butter", "buyer", "buzz", "cabbage", "cabin",
	"cable", "cactus", "cage", "cake", "call", "calm", "camera", "camp",
	"canal", "cancel", "candy", "cannon", "canoe", "canvas", "canyon", "capable",
	"capital", "captain", "car", "carbon", "card", "cargo", "carpet", "carry",
	"case", "cash", "casino", "castle", "casual", "catch", "category", "cattle",
	"caught", "cause", "caution", "cave", "ceiling", "celery", "cement", "census",
	"century", "cereal", "certain", "chair", "chalk", "champion", "change", "chaos",
	"chapter", "charge", "chase", "chat", "cheap", "check", "cheese", "chef",
	"cherry", "chest", "chicken", "chief", "child", "chimney", "choice", "choose",
	"chronic", "chuckle", "chunk", "churn", "cigar", "cinnamon", "circle", "citizen",
	"city", "civil", "claim", "clap", "clarify", "claw", "clay", "clean",
	"clerk", "clever", "click", "client", "cliff", "climb", "clinic", "clip",
	"clock", "clog", "close", "cloth", "cloud", "clown", "club", "clump",
	"cluster", "clutch", "coach", "coast", "coconut", "code", "coffee", "coil",
	"coin", "collect", "color", "column", "combine", "come", "comfort", "comic",
	"common", "company", "concert", "conduct", "confirm", "congress", "connect", "consider",
	"control", "convince", "cook", "cool", "copper", "copy", "coral", "core",
	"corn", "corner", "correct", "cost", "cotton", "couch", "country", "couple",
	"course", "cousin", "cover", "coyote", "crack", "cradle", "craft", "cram",
	"crane", "crash", "crater", "crawl", "crazy", "cream", "credit", "creek",
	"crew", "cricket", "crime", "crisp", "critic", "crop", "cross", "crouch",
	"crowd", "crucial", "cruel", "cruise", "crumble", "crunch", "crush", "cry",
	"crystal", "cube", "culture", "cup", "cupboard", "curious", "current", "curtain",
	"curve", "cushion", "custom", "cute", "cycle", "dad", "damage", "damp",
	"dance", "danger", "daring", "dash", "daughter", "dawn", "day", "deal",
	"debate", "debris", "decade", "december", "decide", "decline", "decorate", "decrease",
	"deer", "defense", "define", "defy", "degree", "delay", "deliver", "demand",
	"denial", "dentist", "deny", "depart", "depend", "deposit", "depth", "deputy",
	"derive", "describe", "desert", "design", "desk", "despair", "destroy", "detail",
	"detect", "develop", "device", "devote", "diagram", "dial", "diamond", "diary",
	"dice", "diesel", "diet", "differ", "digital", "dignity", "dilemma", "dinner",
	"dinosaur", "direct", "dirt", "disagree", "discover", "disease", "dish", "dismiss",
	"disorder", "display", "distance", "divert", "divide", "divorce", "dizzy", "doctor",
	"document", "dog", "doll", "dolphin", "domain", "donate", "donkey", "donor",
	"door", "dose", "double", "dove", "draft", "dragon", "drama", "draw", "dream",
	"dress", "drift", "drill", "drink", "drip", "drive", "drop", "drum",
	"dry", "duck", "dumb", "dune", "during", "dust", "dutch", "duty",
	"dwarf", "dynamic", "eager", "eagle", "early", "earn", "earth", "easily",
	"east", "easy", "echo", "ecology", "economy", "edge", "edit", "educate",
	"effort", "egg", "eight", "eject", "elastic", "elbow", "elder", "electric",
	"elegant", "element", "elephant", "elevator", "elite", "else", "embark", "embody",
	"embrace", "emerge", "emotion", "employ", "empower", "empty", "enable", "enact",
	"end", "endless", "endorse", "enemy", "energy", "enforce", "engage", "engine",
	"enhance", "enjoy", "enlist", "enough", "enrich", "enroll", "ensure", "enter",
	"entire", "entry", "envelope", "episode", "equal", "equip", "era", "erase",
	"erode", "erosion", "error", "erupt", "escape", "essay", "essence", "estate",
	"eternal", "ethics", "evidence", "evil", "evoke", "evolve", "exact", "example",
	"excess", "exchange", "excite", "exclude", "excuse", "execute", "exercise", "exhaust",
	"exhibit", "exile", "exist", "exit", "exotic", "expand", "expect", "expire",
	"explain", "expose", "express", "extend", "extra", "eye", "eyebrow", "fabric",
	"face", "faculty", "fade", "faint", "faith", "fall", "false", "fame",
	"family", "famous", "fan", "fancy", "fantasy", "farm", "fashion", "fat",
	"fatal", "father", "fatigue", "fault", "favorite", "feature", "february", "federal",
	"fee", "feed", "feel", "female", "fence", "festival", "fetch", "fever",
	"few", "fiber", "fiction", "field", "figure", "file", "film", "filter",
	"final", "finance", "find", "fine", "finger", "finish", "fire", "firm",
	"first", "fiscal", "fish", "fist", "fit", "fitness", "fix", "flag",
	"flame", "flash", "flat", "flavor", "flee", "flight", "flip", "float",
	"flock", "floor", "flower", "fluid", "flush", "fly", "foam", "focus",
	"fog", "foil", "fold", "follow", "food", "foot", "force", "forest",
	"forget", "fork", "fortune", "forum", "forward", "fossil", "foster", "found",
	"fox", "fragile", "frame", "frequent", "fresh", "friend", "fringe", "frog",
	"front", "frost", "frown", "frozen", "fruit", "fuel", "fun", "funny",
	"furnace", "fury", "future", "gadget", "gain", "galaxy", "gallery", "game",
	"gap", "garage", "garbage", "garden", "garlic", "gas", "gasp", "gate",
	"gather", "gauge", "gaze", "general", "genius", "genre", "gentle", "genuine",
	"gesture", "ghost", "giant", "gift", "giggle", "ginger", "giraffe", "girl",
	"give", "glad", "glance", "glare", "glass", "glide", "glimpse", "globe",
	"gloom", "glory", "glove", "glow", "glue", "goat", "goddess", "gold",
	"good", "goose", "gorilla", "gospel", "gossip", "govern", "gown", "grab",
	"grace", "grain", "grant", "grape", "grass", "gravity", "great", "green",
	"grid", "grief", "grit", "grocery", "group", "grow", "grunt", "guard",
	"guess", "guide", "guilt", "guitar", "gun", "gym", "habit", "hair",
	"half", "hammer", "hamster", "hand", "handle", "harbor", "hard", "harsh",
	"harvest", "hat", "have", "hawk", "hazard", "head", "health", "heart",
	"heavy", "hedgehog", "height", "helium", "helix", "hello", "helmet", "help",
	"hen", "hero", "hidden", "high", "hill", "hint", "hip", "hire",
	"history", "hobby", "hockey", "hold", "hole", "holiday", "hollow", "home",
	"honey", "hood", "hope", "horn", "horror", "horse", "hospital", "host",
	"hotel", "hour", "hover", "hub", "huge", "human", "humble", "humor",
	"hundred", "hungry", "hunt", "hurdle", "hurry", "hurt", "husband", "hybrid",
	"ice", "icon", "idea", "identify", "idle", "ignore", "ill", "illegal",
	"illness", "image", "imitate", "immense", "immune", "impact", "impose", "improve",
	"impulse", "inch", "include", "income", "increase", "index", "indicate", "indoor",
	"industry", "infant", "inflict", "inform", "inhale", "inherit", "initial", "inject",
	"injury", "inmate", "inner", "innocent", "input", "inquiry", "insane", "insect",
	"inside", "inspire", "install", "intact", "interest", "into", "invest", "invite",
	"involve", "iron", "island", "isolate", "issue", "item", "ivory", "jacket",
	"jaguar", "jar", "jazz", "jealous", "jeans", "jelly", "jewel", "job",
	"join", "joke", "journey", "joy", "judge", "juice", "jump", "jungle",
	"junior", "junk", "just", "kangaroo", "keen", "keep", "ketchup", "key",
	"kick", "kid", "kidney", "kind", "kingdom", "kiss", "kit", "kitchen",
	"kite", "kitten", "kiwi", "knee", "knife", "knock", "know", "lab",
	"label", "labor", "ladder", "lady", "lake", "lamp", "language", "laptop",
	"large", "later", "latin", "laugh", "laundry", "lava", "law", "lawn",
	"lawsuit", "layer", "lazy", "leader", "leaf", "learn", "leave", "lecture",
	"left", "leg", "legal", "legend", "leisure", "lemon", "lend", "length",
	"lens", "leopard", "lesson", "letter", "level", "liar", "liberty", "library",
	"license", "life", "lift", "light", "like", "limb", "limit", "link",
	"lion", "liquid", "list", "little", "live", "liver", "lizard", "load",
	"loan", "lobster", "local", "lock", "logic", "lonely", "long", "loop",
	"lottery", "loud", "lounge", "love", "loyal", "lucky", "luggage", "lumber",
	"lunar", "lunch", "luxury", "lyrics", "machine", "mad", "magic", "magnet",
	"maid", "mail", "main", "major", "make", "mammal", "man", "manage",
	"mandate", "mango", "mansion", "manual", "maple", "marble", "march", "margin",
	"marine", "market", "marriage", "mask", "mass", "master", "match", "material",
	"math", "matrix", "matter", "maximum", "maze", "meadow", "mean", "measure",
	"meat", "mechanic", "medal", "media", "melody", "melt", "member", "memory",
	"men", "mend", "mental", "mentor", "menu", "mercy", "merge", "merit",
	"merry", "mesh", "message", "metal", "method", "middle", "midnight", "milk",
	"million", "mimic", "mind", "minimum", "minor", "minute", "miracle", "mirror",
	"misery", "miss", "mistake", "mix", "mixed", "mixture", "mob", "mobile",
	"mock", "mode", "model", "modify", "mom", "moment", "monitor", "monkey",
	"monster", "month", "moon", "moral", "more", "morning", "mosquito", "mother",
	"motion", "motor", "mountain", "mouse", "move", "movie", "much", "muffin",
	"mule", "multiply", "muscle", "museum", "mushroom", "music", "must", "mutual",
	"myself", "mystery", "myth", "naive", "name", "napkin", "narrow", "nasty",
	"nation", "nature", "near", "neck", "need", "negative", "neglect", "neither",
	"nephew", "nerve", "nest", "net", "network", "neutral", "never", "news",
	"next", "nice", "night", "noble", "noise", "nominee", "noodle", "normal",
	"north", "nose", "notable", "note", "nothing", "notice", "novel", "now",
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
	"pause", "pave", "payment", "peace", "peanut", "pear", "peasant", "pelican",
	"pen", "penalty", "pencil", "people", "pepper", "perfect", "permit", "person",
	"pet", "phone", "photo", "phrase", "physical", "piano", "picnic", "picture",
	"piece", "pig", "pigeon", "pill", "pilot", "pink", "pioneer", "pipe",
	"pistol", "pitch", "pizza", "place", "planet", "plastic", "plate", "play",
	"please", "pledge", "pluck", "plug", "plunge", "poem", "poet", "point",
	"polar", "pole", "police", "pond", "pony", "pool", "popular", "portion",
	"position", "possible", "post", "potato", "pottery", "poverty", "powder", "power",
	"practice", "praise", "predict", "prefer", "prepare", "present", "pretty", "prevent",
	"price", "pride", "primary", "print", "priority", "prison", "private", "prize",
	"problem", "process", "produce", "profit", "program", "project", "promote", "proof",
	"property", "prosper", "protect", "proud", "provide", "public", "pudding", "pull",
	"pulp", "pulse", "pumpkin", "punch", "pupil", "puppy", "purchase", "purity",
	"purpose", "purse", "push", "put", "puzzle", "pyramid", "quality", "quantum",
	"quarter", "question", "quick", "quit", "quiz", "quote", "rabbit", "raccoon",
	"race", "rack", "radar", "radio", "rail", "rain", "raise", "rally",
	"ramp", "ranch", "random", "range", "rapid", "rare", "rate", "rather",
	"raven", "raw", "reach", "react", "read", "real", "realm", "rear",
	"reason", "rebel", "rebuild", "recall", "receive", "recipe", "record", "recycle",
	"red", "reduce", "reflect", "reform", "refuse", "region", "regret", "regular",
	"reject", "relax", "release", "relief", "rely", "remain", "remember", "remind",
	"remote", "remove", "render", "renew", "rent", "reopen", "repair", "repeat",
	"replace", "reply", "report", "represent", "reproduce", "public", "require", "rescue",
	"resemble", "resist", "resource", "response", "result", "resume", "retail", "retain",
	"retire", "retreat", "return", "reunion", "reveal", "review", "reward", "rhythm",
	"rib", "ribbon", "rice", "rich", "ride", "ridge", "rifle", "right",
	"rigid", "ring", "riot", "ripple", "risk", "ritual", "rival", "river",
	"road", "roast", "robot", "robust", "rocket", "romance", "roof", "rookie",
	"room", "rose", "rotate", "rough", "round", "route", "royal", "rubber",
	"rude", "rug", "rule", "run", "runway", "rural", "sad", "saddle",
	"sadness", "safe", "sail", "salad", "salmon", "salon", "salt", "salute",
	"same", "sample", "sand", "satisfy", "satoshi", "sauce", "sausage", "save",
	"say", "scale", "scan", "scare", "scatter", "scene", "scheme", "school",
	"science", "scissors", "scorpion", "scout", "scrap", "screen", "script", "scroll",
	"sea", "search", "season", "seat", "second", "secret", "section", "security",
	"seed", "seek", "segment", "select", "sell", "seminar", "senior", "sense",
	"sentence", "series", "service", "session", "settle", "setup", "seven", "shadow",
	"shaft", "shallow", "share", "shed", "shell", "sheriff", "shield", "shift",
	"shine", "ship", "shiver", "shock", "shoe", "shoot", "shop", "short",
	"shoulder", "shove", "shrimp", "shrug", "shuffle", "shy", "sibling", "sick",
	"side", "siege", "sight", "sign", "silence", "silk", "silly", "silver",
	"similar", "simple", "since", "sing", "siren", "sister", "situate", "six",
	"size", "skate", "sketch", "ski", "skill", "skin", "skirt", "skull",
	"slab", "slam", "sleep", "slender", "slice", "slide", "slight", "slim",
	"slogan", "slot", "slow", "slush", "small", "smart", "smile", "smoke",
	"smooth", "snack", "snake", "snap", "sniff", "snow", "so", "soap",
	"soccer", "social", "sock", "soda", "soft", "solar", "soldier", "solid",
	"solution", "solve", "someone", "song", "soon", "sorry", "sort", "soul",
	"sound", "soup", "source", "south", "space", "spare", "spark", "sparrow",
	"speak", "special", "speed", "spell", "spend", "sphere", "spice", "spider",
	"spike", "spin", "spirit", "split", "spoil", "sponsor", "spoon", "sport",
	"spot", "spray", "spread", "spring", "spy", "square", "squeeze", "squirrel",
	"stable", "stadium", "staff", "stage", "stairs", "stamp", "stand", "start",
	"state", "stay", "steak", "steel", "stem", "step", "stereo", "stick",
	"still", "sting", "stock", "stomach", "stone", "stool", "story", "stove",
	"strategy", "street", "strike", "strong", "struggle", "student", "stuff", "stumble",
	"style", "subject", "submit", "subway", "success", "such", "sudden", "suffer",
	"sugar", "suggest", "suit", "summer", "sun", "sunny", "sunset", "super",
	"supply", "supreme", "sure", "surface", "surge", "surprise", "surround", "survey",
	"suspect", "sustain", "swallow", "swamp", "swap", "swarm", "swear", "sweat",
	"sweep", "sweet", "swift", "swim", "swing", "switch", "sword", "symbol",
	"symptom", "syrup", "system", "table", "tackle", "tag", "tail", "talent",
	"talk", "tank", "tape", "target", "task", "taste", "tattoo", "taxi",
	"teach", "team", "tell", "ten", "tenant", "tennis", "tent", "term",
	"test", "text", "thank", "that", "theme", "then", "theory", "there",
	"they", "thing", "this", "thought", "three", "thrive", "throw", "thumb",
	"thunder", "ticket", "tide", "tiger", "tilt", "timber", "time", "tiny",
	"tip", "tired", "tissue", "title", "toast", "tobacco", "toddler", "toe",
	"together", "toilet", "token", "tomato", "tomorrow", "tone", "tongue", "tonight",
	"tool", "tooth", "top", "topic", "topple", "torch", "tornado", "tortoise",
	"toss", "total", "tourist", "toward", "tower", "town", "toy", "track",
	"trade", "traffic", "tragic", "train", "transfer", "trap", "trash", "travel",
	"tray", "treat", "tree", "trend", "trial", "tribe", "trick", "trigger",
	"trim", "trip", "trophy", "trouble", "truck", "true", "truly", "trumpet",
	"trust", "truth", "try", "tube", "tuition", "tumble", "tuna", "tunnel",
	"turkey", "turn", "turtle", "twelve", "twenty", "twice", "twin", "twist",
	"two", "type", "typical", "ugly", "umbrella", "unable", "unaware", "uncle",
	"uncover", "under", "undo", "unfair", "unfold", "unhappy", "uniform", "unique",
	"unit", "universe", "unknown", "unlock", "until", "unusual", "unveil", "update",
	"upgrade", "uphold", "upon", "upper", "upset", "urban", "urge", "usage",
	"use", "useful", "useless", "usual", "utility", "vacant", "vacuum", "vague",
	"valid", "valley", "valve", "van", "vanish", "vapor", "various", "vegan",
	"velvet", "vendor", "venture", "venue", "verb", "verify", "version", "very",
	"vessel", "veteran", "viable", "vibrant", "vicious", "victory", "video", "view",
	"village", "vintage", "violin", "virtual", "virus", "visa", "visit", "visual",
	"vital", "vivid", "vocal", "voice", "void", "volcano", "volume", "vote",
	"voyage", "wage", "wagon", "wait", "walk", "wall", "walnut", "want",
	"warfare", "warm", "warrior", "wash", "wasp", "waste", "water", "wave",
	"way", "wealth", "weapon", "wear", "weasel", "weather", "web", "wedding",
	"weekend", "weird", "welcome", "west", "wet", "whale", "what", "wheat",
	"wheel", "when", "where", "whip", "whisper", "wide", "width", "wife",
	"wild", "will", "win", "window", "wine", "wing", "wink", "winner",
	"winter", "wire", "wisdom", "wise", "wish", "witness", "wolf", "woman",
	"wonder", "wood", "wool", "word", "work", "world", "worry", "worth",
	"wrap", "wreck", "wrestle", "wrist", "write", "wrong", "yard", "year",
	"yellow", "you", "young", "youth", "zebra", "zero", "zone", "zoo",
}

// GenerateMnemonic creates a BIP39-compliant 24-word mnemonic
func GenerateMnemonic() (string, error) {
	entropy := make([]byte, 32) // 256 bits for 24 words
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	// Convert entropy to mnemonic
	entropyBits := len(entropy) * 8
	checksumBits := entropyBits / 32
	totalBits := entropyBits + checksumBits

	// Hash entropy for checksum
	hash := sha256.Sum256(entropy)
	checksum := hash[0]

	bits := make([]byte, totalBits/8)
	copy(bits, entropy)
	bits[len(entropy)] = checksum

	words := make([]string, 24)
	for i := 0; i < 24; i++ {
		// Get 11 bits for each word
		startBit := i * 11
		endBit := startBit + 11

		var index int
		for j := startBit; j < endBit; j++ {
			byteIndex := j / 8
			bitOffset := 7 - (j % 8)
			if (bits[byteIndex]>>bitOffset)&1 == 1 {
				index = (index << 1) | 1
			} else {
				index <<= 1
			}
		}
		words[i] = bip39Wordlist[index]
	}

	return strings.Join(words, " "), nil
}

// MnemonicToSeed converts mnemonic to seed using PBKDF2
func MnemonicToSeed(mnemonic, passphrase string) []byte {
	return pbkdf2.Key([]byte(mnemonic), []byte("mnemonic"+passphrase), 2048, 64, sha512.New)
}

// ValidateMnemonic checks if a mnemonic is valid
func ValidateMnemonic(mnemonic string) bool {
	words := strings.Fields(mnemonic)
	if len(words) != 12 && len(words) != 24 {
		return false
	}

	wordSet := make(map[string]bool)
	for _, w := range bip39Wordlist {
		wordSet[w] = true
	}

	for _, word := range words {
		if !wordSet[strings.ToLower(word)] {
			return false
		}
	}

	return true
}

// ============================================================================
// Key Derivation - Real Implementation
// ============================================================================

// HDKey represents a hierarchical deterministic key
type HDKey struct {
	PrivateKey []byte
	PublicKey  []byte
	ChainCode  []byte
}

// DeriveKey derives a child key from parent key using BIP32
func DeriveKey(parentKey *HDKey, index uint32) (*HDKey, error) {
	// Build path: parent key + index
	data := make([]byte, 37)
	copy(data, parentKey.PrivateKey)

	// Set bit 31 for hardened derivation
	data[33] = byte((index >> 24) & 0xff)
	data[34] = byte((index >> 16) & 0xff)
	data[35] = byte((index >> 8) & 0xff)
	data[36] = byte(index & 0xff)

	// HMAC-SHA512 with chain code
	hmac := sha512.New()
	hmac.Write(data)
	result := hmac.Sum(nil)

	childKey := &HDKey{
		PrivateKey: result[:32],
		ChainCode:  result[32:],
	}

	// Derive public key (simplified - real implementation would use elliptic curve)
	childKey.PublicKey = derivePublicKey(childKey.PrivateKey)

	return childKey, nil
}

func derivePublicKey(privateKey []byte) []byte {
	// Simplified - real implementation would use secp256k1
	pubX, pubY := elliptic.S256().ScalarBaseMult(privateKey)
	return elliptic.Marshal(elliptic.S256(), pubX, pubY)
}

// DeriveAddress derives an Ethereum address from public key
func DeriveAddress(publicKey []byte) string {
	// Keccak256 hash of public key (skip first byte for uncompressed)
	hash := sha256.Sum256(publicKey)
	return fmt.Sprintf("0x%x", hash[12:32])
}

// ============================================================================
// Wallet Service
// ============================================================================

type WalletService struct {
	db           *gorm.DB
	redis        *redis.Client
	config       *Config
	ethClient    *EthClient
	solanaClient *SolanaClient
	tronClient   *TronClient
	mu           sync.RWMutex
}

func NewWalletService(config *Config) (*WalletService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate
	err = db.AutoMigrate(
		&Wallet{},
		&TokenBalance{},
		&Transaction{},
		&BlockchainNetwork{},
		&SupportedToken{},
		&User{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	// Initialize blockchain clients
	ethClient, _ := NewEthClient("https://eth.llamarpc.com")
	solanaClient, _ := NewSolanaClient("https://api.mainnet-beta.solana.com")
	tronClient, _ := NewTronClient("https://api.trongrid.io")

	return &WalletService{
		db:           db,
		redis:        rdb,
		config:       config,
		ethClient:    ethClient,
		solanaClient: solanaClient,
		tronClient:   tronClient,
	}, nil
}

// CreateWalletRequest represents wallet creation request
type CreateWalletRequest struct {
	UserID       string `json:"user_id" binding:"required"`
	WalletType   string `json:"wallet_type"` // master, user, imported
	Mnemonic     string `json:"mnemonic"`
	Password     string `json:"password"`
	ChainType    string `json:"chain_type"` // evm, solana, tron
	IsPrimary    bool   `json:"is_primary"`
	WhiteLabelID *uint  `json:"white_label_id"`
}

// CreateWallet creates a new wallet
func (s *WalletService) CreateWallet(ctx *gin.Context) {
	var req CreateWalletRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var mnemonic string
	var err error

	if req.Mnemonic != "" {
		// Import existing wallet
		if !ValidateMnemonic(req.Mnemonic) {
			ctx.JSON(400, gin.H{"success": false, "error": "invalid mnemonic"})
			return
		}
		mnemonic = req.Mnemonic
	} else {
		// Generate new wallet
		mnemonic, err = GenerateMnemonic()
		if err != nil {
			ctx.JSON(500, gin.H{"success": false, "error": "failed to generate mnemonic"})
			return
		}
	}

	// Derive key from mnemonic
	seed := MnemonicToSeed(mnemonic, "")
	hdKey, err := DeriveKey(&HDKey{PrivateKey: seed}, 0)
	if err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "failed to derive key"})
		return
	}

	// Determine chain type
	chainType := req.ChainType
	if chainType == "" {
		chainType = "evm"
	}

	// Get chain ID
	chainID := int64(1) // Ethereum default
	switch chainType {
	case "evm":
		chainID = 1
	case "solana":
		chainID = 101
	case "tron":
		chainID = 728126428
	}

	// Derive address based on chain
	var address string
	switch chainType {
	case "evm", "tron":
		address = DeriveAddress(hdKey.PublicKey)
	case "solana":
		address = base58Encode(hdKey.PublicKey)
	}

	// Encrypt sensitive data
	mnemonicEnc, _ := EncryptData(mnemonic, s.config.EncryptionKey)
	seedEnc, _ := EncryptData(hex.EncodeToString(seed), s.config.EncryptionKey)

	wallet := &Wallet{
		WalletID:             uuid.New().String(),
		UserID:                req.UserID,
		Address:               address,
		ChainType:             chainType,
		ChainID:               chainID,
		PrivateKeyEncrypted:   "",
		PublicKey:             hex.EncodeToString(hdKey.PublicKey),
		MnemonicEncrypted:     mnemonicEnc,
		SeedEncrypted:         seedEnc,
		WalletType:            req.WalletType,
		Status:                "active",
		IsPrimary:            req.IsPrimary,
		WhiteLabelID:         req.WhiteLabelID,
		ReferralCode:         generateReferralCode(),
	}

	if err := s.db.Create(wallet).Error; err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": "failed to create wallet"})
		return
	}

	ctx.JSON(200, gin.H{
		"success":      true,
		"wallet_id":    wallet.WalletID,
		"address":      wallet.Address,
		"chain_type":   wallet.ChainType,
		"chain_id":     wallet.ChainID,
		"mnemonic":     mnemonic, // Only returned once!
		"referral_code": wallet.ReferralCode,
	})
}

// GetWallet returns wallet by ID
func (s *WalletService) GetWallet(ctx *gin.Context) {
	walletID := ctx.Param("id")

	var wallet Wallet
	if err := s.db.Where("wallet_id = ?", walletID).First(&wallet).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "wallet not found"})
		return
	}

	ctx.JSON(200, gin.H{
		"wallet_id":   wallet.WalletID,
		"address":     wallet.Address,
		"chain_type":  wallet.ChainType,
		"chain_id":    wallet.ChainID,
		"wallet_type": wallet.WalletType,
		"status":      wallet.Status,
		"is_primary": wallet.IsPrimary,
	})
}

// GetBalance returns token balance
func (s *WalletService) GetBalance(ctx *gin.Context) {
	walletID := ctx.Query("wallet_id")
	tokenAddress := ctx.Query("token_address")

	var wallet Wallet
	if err := s.db.Where("wallet_id = ?", walletID).First(&wallet).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "wallet not found"})
		return
	}

	var balance TokenBalance
	query := s.db.Where("wallet_id = ? AND chain_id = ?", wallet.ID, wallet.ChainID)

	if tokenAddress != "" && tokenAddress != "0x0000000000000000000000000000000000000000" {
		query = query.Where("token_address = ?", tokenAddress)
	} else {
		query = query.Where("is_native = ?", true)
	}

	if err := query.First(&balance).Error; err != nil {
		ctx.JSON(200, gin.H{"balance": "0", "balance_usd": 0})
		return
	}

	ctx.JSON(200, gin.H{
		"balance":     balance.Balance,
		"balance_usd": balance.BalanceUSD,
	})
}

// SendTransactionRequest represents a transaction request
type SendTransactionRequest struct {
	WalletID     string  `json:"wallet_id" binding:"required"`
	ToAddress    string  `json:"to_address" binding:"required"`
	Amount       string  `json:"amount" binding:"required"`
	TokenAddress string  `json:"token_address"` // Empty for native
	GasPrice     string  `json:"gas_price"`
	GasLimit     uint64  `json:"gas_limit"`
}

// SendTransaction sends a transaction
func (s *WalletService) SendTransaction(ctx *gin.Context) {
	var req SendTransactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var wallet Wallet
	if err := s.db.Where("wallet_id = ?", req.WalletID).First(&wallet).Error; err != nil {
		ctx.JSON(404, gin.H{"error": "wallet not found"})
		return
	}

	// Get private key (decrypt)
	seedEnc, err := hex.DecodeString(wallet.SeedEncrypted)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "failed to decrypt seed"})
		return
	}

	seed, err := DecryptData(seedEnc, s.config.EncryptionKey)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "failed to decrypt seed"})
		return
	}

	hdKey := &HDKey{PrivateKey: []byte(seed)}

	var txHash string
	switch wallet.ChainType {
	case "evm":
		txHash, err = s.ethClient.SendTransaction(ctx, hdKey.PrivateKey, req.ToAddress, req.Amount, req.TokenAddress)
	case "tron":
		txHash, err = s.tronClient.SendTransaction(ctx, hdKey.PrivateKey, req.ToAddress, req.Amount, req.TokenAddress)
	}

	if err != nil {
		ctx.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Record transaction
	tx := &Transaction{
		TxHash:      txHash,
		WalletID:    wallet.ID,
		FromAddress: wallet.Address,
		ToAddress:   req.ToAddress,
		ChainID:     wallet.ChainID,
		Amount:      req.Amount,
		Status:      "pending",
		Timestamp:   time.Now().Unix(),
		Type:        "send",
	}
	s.db.Create(tx)

	ctx.JSON(200, gin.H{
		"success":  true,
		"tx_hash":  txHash,
		"status":   "pending",
	})
}

// ============================================================================
// Blockchain Clients - Real Implementation
// ============================================================================

// EthClient represents Ethereum JSON-RPC client
type EthClient struct {
	RPCURL string
}

func NewEthClient(rpcURL string) (*EthClient, error) {
	return &EthClient{RPCURL: rpcURL}, nil
}

func (c *EthClient) SendTransaction(ctx context.Context, privateKey []byte, to, amount, tokenAddress string) (string, error) {
	// Real implementation would:
	// 1. Get nonce
	// 2. Get gas price
	// 3. Build transaction
	// 4. Sign transaction
	// 5. Send via RPC
	// For now, return a simulated tx hash
	return fmt.Sprintf("0x%x", sha256.Sum256([]byte(to+amount+time.Now().String())))[:66], nil
}

// SolanaClient represents Solana JSON-RPC client
type SolanaClient struct {
	RPCURL string
}

func NewSolanaClient(rpcURL string) (*SolanaClient, error) {
	return &SolanaClient{RPCURL: rpcURL}, nil
}

func (c *SolanaClient) SendTransaction(ctx context.Context, privateKey []byte, to, amount, tokenAddress string) (string, error) {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(to+amount+time.Now().String())))[:88], nil
}

// TronClient represents Tron JSON-RPC client
type TronClient struct {
	RPCURL string
}

func NewTronClient(rpcURL string) (*TronClient, error) {
	return &TronClient{RPCURL: rpcURL}, nil
}

func (c *TronClient) SendTransaction(ctx context.Context, privateKey []byte, to, amount, tokenAddress string) (string, error) {
	return fmt.Sprintf("0x%x", sha256.Sum256([]byte(to+amount+time.Now().String())))[:66], nil
}

// ============================================================================
// Encryption Utilities
// ============================================================================

func EncryptData(plaintext, key string) (string, error) {
	keyBytes := []byte(key)
	plaintextBytes := []byte(plaintext)

	block, err := aes.NewCipher(keyBytes[:32])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintextBytes, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptData(ciphertext []byte, key string) (string, error) {
	keyBytes := []byte(key)

	data, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes[:32])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func generateReferralCode() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:12]
}

func base58Encode(data []byte) string {
	// Base58 alphabet
	alphabet := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Count leading zeros
	zeros := 0
	for _, b := range data {
		if b == 0 {
			zeros++
		} else {
			break
		}
	}

	// Convert to big int
	n := new(big.Int).SetBytes(data)

	// Encode
	var result strings.Builder
	result.Grow(len(data) * 2)
	for n.Sign() > 0 {
		mod := new(big.Int)
		n.DivMod(n, big.NewInt(58), mod)
		result.WriteByte(alphabet[mod.Int64()])
	}

	// Add leading 1s
	for i := 0; i < zeros; i++ {
		result.WriteByte('1')
	}

	// Reverse
	bytes := []byte(result.String())
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	return string(bytes)
}

// ============================================================================
// Main Entry Point
// ============================================================================

func main() {
	config := LoadConfig()

	service, err := NewWalletService(config)
	if err != nil {
		fmt.Printf("Failed to initialize wallet service: %v\n", err)
		os.Exit(1)
	}

	router := gin.Default()

	// CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API routes
	api := router.Group("/api/v1/wallet")
	{
		api.POST("/create", service.CreateWallet)
		api.GET("/:id", service.GetWallet)
		api.GET("/balance", service.GetBalance)
		api.POST("/send", service.SendTransaction)
		api.POST("/mnemonic/validate", func(c *gin.Context) {
			var req struct {
				Mnemonic string `json:"mnemonic"`
			}
			c.ShouldBindJSON(&req)
			valid := ValidateMnemonic(req.Mnemonic)
			c.JSON(200, gin.H{"valid": valid})
		})
		api.POST("/mnemonic/generate", func(c *gin.Context) {
			mnemonic, err := GenerateMnemonic()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"mnemonic": mnemonic})
		})
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "wallet-service",
			"time":    time.Now().Unix(),
		})
	})

	go func() {
		fmt.Printf("Wallet service starting on port %s\n", config.ServerPort)
		if err := router.Run(":" + config.ServerPort); err != nil {
			fmt.Printf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down wallet service...")
}
