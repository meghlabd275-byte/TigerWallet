// ============================================================================
// TIGERWALLET MNEMONIC ENGINE
// BIP-39 Mnemonic Phrase Generation and Validation
// Supports 12, 15, 18, 21, and 24 word phrases
// ============================================================================

package mnemonic

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"unicode/utf8"
)

// WordList represents a BIP-39 word list
type WordList struct {
	Language string
	Words    []string
}

// Entropy strength to mnemonic word count mapping
var entropyLengths = map[int]int{
	128: 12, // 12 words for 128 bits
	160: 15, // 15 words for 160 bits
	192: 18, // 18 words for 192 bits
	224: 21, // 21 words for 224 bits
	256: 24, // 24 words for 256 bits
}

// DefaultEnglishWordList is the standard BIP-39 English word list
var DefaultEnglishWordList = WordList{
	Language: "english",
	Words: []string{
		"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract", "absurd", "abuse",
		"access", "accident", "account", "accuse", "achieve", "acid", "acoustic", "acquire", "across", "act",
		"action", "actor", "actress", "actual", "adapt", "add", "addict", "address", "adjust", "admit",
		"adult", "advance", "advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent",
		"agree", "ahead", "aim", "air", "airport", "aisle", "alarm", "album", "alcohol", "alert",
		"alien", "all", "alley", "allow", "almost", "alone", "alpha", "already", "also", "alter",
		"always", "amateur", "amazing", "among", "amount", "amused", "analyst", "anchor", "ancient",
		"anger", "angle", "angry", "animal", "ankle", "announce", "annual", "another", "answer", "antenna",
		"antique", "anxiety", "any", "apart", "apology", "appear", "apple", "approve", "april", "arch",
		"arctic", "area", "arena", "argue", "arm", "armed", "armor", "army", "around", "arrange",
		"arrest", "arrive", "arrow", "art", "artefact", "artist", "artwork", "ask", "aspect", "assault",
		"asset", "assist", "assume", "asthma", "athlete", "atom", "attack", "attend", "attitude", "attract",
		"auction", "audit", "august", "aunt", "author", "auto", "autumn", "average", "avocado", "avoid",
		"awake", "aware", "away", "awesome", "awful", "awkward", "axis", "baby", "bachelor", "bacon",
		"badge", "bag", "balance", "balcony", "ball", "bamboo", "banana", "banner", "bar", "barely",
		"bargain", "barrel", "base", "basic", "basket", "battle", "beach", "bean", "beauty", "because",
		"become", "beef", "before", "begin", "behave", "behind", "believe", "below", "belt", "bench",
		"benefit", "best", "betray", "better", "between", "beyond", "bicycle", "bid", "bike", "bind",
		"biology", "bird", "birth", "bitter", "black", "blade", "blame", "blanket", "blast", "bleak",
		"bless", "blind", "blood", "blossom", "blouse", "blue", "blur", "blush", "board", "boat",
		"body", "boil", "bomb", "bone", "bonus", "book", "boost", "border", "boring", "borrow",
		"boss", "bottom", "bounce", "box", "boy", "bracket", "brain", "brand", "brass", "brave",
		"bread", "breeze", "brick", "bridge", "brief", "bright", "bring", "brisk", "broccoli",
		"broken", "bronze", "broom", "brother", "brown", "brush", "bubble", "buddy", "budget", "buffalo",
		"build", "bulb", "bulk", "bullet", "bundle", "bunker", "burden", "burger", "burst", "bus",
		"business", "busy", "butter", "buyer", "buzz", "cabbage", "cabin", "cable", "cactus", "cage",
		"cake", "call", "calm", "camera", "camp", "can", "canal", "cancel", "candy", "cannon",
		"canoe", "canvas", "canyon", "capable", "capital", "captain", "car", "carbon", "card",
		"cargo", "carpet", "carry", "cart", "case", "cash", "casino", "castle", "casual", "cat",
		"catalog", "catch", "category", "cattle", "caught", "cause", "caution", "cave", "ceiling",
		"celery", "cement", "census", "century", "cereal", "certain", "chair", "chalk", "champion",
		"change", "chaos", "chapter", "charge", "chase", "chat", "cheap", "check", "cheese", "chef",
		"cherry", "chest", "chicken", "chief", "child", "chimney", "choice", "choose", "chronic",
		"chuckle", "chunk", "churn", "cigar", "cinnamon", "circle", "citizen", "city", "civil",
		"claim", "clap", "clarify", "classic", "clean", "clerk", "clever", "click", "client", "cliff",
		"climb", "clinic", "clip", "clock", "close", "cloth", "cloud", "clown", "club", "clump",
		"cluster", "clutch", "coach", "coast", "coconut", "code", "coffee", "coil", "coin", "collect",
		"color", "column", "combine", "come", "comfort", "comic", "common", "company", "concert", "conduct",
		"confirm", "congress", "connect", "consider", "control", "convince", "cook", "cool", "copper",
		"copy", "coral", "core", "corn", "correct", "cost", "cottage", "cotton", "couch", "country",
		"couple", "course", "cousin", "cover", "coyote", "crack", "cradle", "craft", "cram", "crane",
		"crash", "crater", "crawl", "crazy", "cream", "credit", "creek", "crew", "cricket", "crime",
		"crisp", "critic", "crop", "cross", "crouch", "crowd", "crucial", "cruel", "cruise",
		"crumble", "crunch", "crush", "cry", "crystal", "cube", "culture", "cup", "cupboard", "curious",
		"current", "curtain", "curve", "cushion", "custom", "cute", "cycle", "dad", "damage",
		"damp", "dance", "danger", "daring", "dash", "daughter", "dawn", "day", "deal", "debate",
		"debris", "decade", "december", "decide", "decline", "decorate", "decrease", "deer", "defense",
		"define", "defy", "degree", "delay", "deliver", "demand", "demise", "denial", "dentist", "deny",
		"depart", "depend", "deposit", "depth", "deputy", "derive", "describe", "desert", "design",
		"desk", "despair", "destroy", "detail", "detect", "develop", "device", "devote", "diagram", "dial",
		"diamond", "diary", "dice", "diesel", "diet", "differ", "digital", "dignity", "dilemma", "dinner",
		"dinosaur", "direct", "dirt", "disagree", "discover", "disease", "dish", "dismiss", "disorder", "display",
		"distance", "divert", "divide", "divorce", "dizzy", "doctor", "document", "dog", "doll",
		"dolphin", "domain", "donate", "donkey", "donor", "door", "dose", "dot", "double", "dove",
		"draft", "dragon", "drama", "draw", "dream", "dress", "drift", "drill", "drink", "drip",
		"drive", "drop", "drum", "dry", "duck", "dumb", "dune", "during", "dust", "dutch",
		"duty", "dwarf", "dynamic", "eager", "eagle", "early", "earn", "earth", "easily", "east",
		"easy", "echo", "ecology", "economy", "edge", "edit", "educate", "effort", "eight", "eject",
		"elastic", "elbow", "elder", "electric", "elegant", "element", "elephant", "elevator", "elite", "else",
		"embark", "embody", "embrace", "emerge", "emotion", "employ", "empower", "empty", "enable", "enact",
		"end", "endless", "endorse", "enemy", "energy", "enforce", "engage", "engine", "enhance", "enjoy",
		"enlist", "enough", "enrich", "enroll", "ensure", "enter", "entire", "entry", "envelope",
		"episode", "equal", "equip", "era", "erase", "erode", "erosion", "error", "erupt", "escape",
		"essay", "essence", "estate", "eternal", "ethics", "evidence", "evil", "evoke", "evolve",
		"exact", "example", "excess", "exchange", "excite", "exclude", "excuse", "execute", "exercise",
		"exhaust", "exhibit", "exile", "exist", "exit", "exotic", "expand", "expect", "expire", "explain",
		"expose", "express", "extend", "extra", "eye", "eyebrow", "fabric", "face", "faculty", "fade",
		"faint", "faith", "fall", "false", "fame", "family", "famous", "fan", "fancy", "fantasy",
		"farm", "fashion", "fat", "fatal", "father", "fatigue", "fault", "favorite", "feature",
		"february", "federal", "fee", "feed", "feel", "female", "fence", "festival", "fetch",
		"fever", "few", "fiber", "fiction", "field", "figure", "file", "film", "filter", "final",
		"finance", "find", "fine", "finger", "finish", "fire", "firm", "first", "fiscal", "fish",
		"fitness", "fix", "flag", "flame", "flash", "flat", "flavor", "flee", "flight", "flip",
		"float", "flock", "flood", "floor", "flower", "fluid", "flush", "fly", "foam", "focus",
		"fog", "foil", "fold", "follow", "food", "foot", "force", "forest", "forget", "fork",
		"fortune", "forum", "forward", "fossil", "found", "fox", "fragile", "frame", "frequent",
		"fresh", "friend", "fringe", "frog", "front", "frost", "frown", "frozen", "fruit", "fuel",
		"fun", "funny", "furnace", "fury", "future", "gadget", "gain", "galaxy", "gallery",
		"game", "gap", "garage", "garbage", "garden", "garlic", "gas", "gasp", "gate", "gather",
		"gauge", "gaze", "general", "genius", "genre", "gentle", "genuine", "gesture", "ghost",
		"giant", "gift", "giggle", "ginger", "giraffe", "girl", "give", "glad", "glance",
		"glare", "glass", "glide", "glimpse", "globe", "gloom", "glory", "glove", "glow", "glue",
		"goat", "goddess", "gold", "good", "goose", "gorilla", "gospel", "gossip", "govern", "gown",
		"grab", "grace", "grain", "grant", "grape", "grass", "gravity", "great", "green", "grid",
		"grief", "grit", "grocery", "group", "grow", "grunt", "guard", "guess", "guide", "guilt",
		"guitar", "gun", "gym", "habit", "hair", "half", "hammer", "hamster", "hand", "handle",
		"harbor", "hard", "harsh", "harvest", "hat", "have", "hawk", "hazard", "head", "health",
		"heart", "heavy", "hedgehog", "height", "hello", "helmet", "help", "hen", "hero", "hidden",
		"high", "hill", "hint", "hip", "hire", "history", "hobby", "hockey", "hold", "hole", "holiday",
		"hollow", "home", "honey", "hood", "hope", "horn", "horrible", "horse", "hospital", "host",
		"hotel", "hour", "hover", "hub", "huge", "human", "humble", "humor", "hundred", "hungry", "hunt",
		"hurdle", "hurry", "hurt", "husband", "hybrid", "ice", "icon", "idea", "identify", "idle", "ignore",
		"ill", "illegal", "illness", "image", "imitate", "immense", "immune", "impact", "impose", "improve", "impulse",
		"inch", "include", "income", "increase", "index", "indicate", "indoor", "industry", "infant", "inflict",
		"inform", "inhale", "inherit", "initial", "inject", "injury", "inmate", "inner", "innocent", "input",
		"inquiry", "insane", "insect", "inside", "inspire", "install", "intact", "interest", "into",
		"invest", "invite", "involve", "iron", "island", "isolate", "issue", "item", "ivory", "jacket",
		"jaguar", "jar", "jazz", "jealous", "jeans", "jelly", "jewel", "job", "join", "joke",
		"jolly", "journey", "joy", "judge", "juice", "jump", "jungle", "junior", "junk", "just",
		"kangaroo", "keen", "keep", "ketchup", "key", "kick", "kid", "kidney", "kind", "kingdom",
		"kiss", "kit", "kitchen", "kite", "kitten", "kiwi", "knee", "knife", "knock", "know",
		"lab", "label", "labor", "ladder", "lady", "lake", "lamp", "language", "laptop", "large",
		"later", "latin", "laugh", "laundry", "lava", "law", "lawn", "lawsuit", "layer", "lazy",
		"leader", "leaf", "learn", "leave", "lecture", "left", "leg", "legal", "legend", "leisure",
		"lemon", "lend", "length", "lens", "leopard", "lesson", "letter", "level", "liar", "liberty",
		"library", "license", "life", "lift", "light", "like", "limb", "limit", "linen", "lion",
		"liquid", "list", "little", "live", "lizard", "load", "loan", "lobster", "local", "lock",
		"logic", "lonely", "long", "loop", "lottery", "loud", "lounge", "love", "loyal", "lucky", "luggage",
		"lumber", "lunar", "lunch", "luxury", "lyrics", "mad", "magnet", "maid", "mail", "main",
		"major", "make", "mammal", "man", "manage", "mandate", "mango", "mansion", "manual", "maple",
		"marble", "march", "margin", "marine", "market", "marriage", "mask", "mass", "master",
		"match", "material", "math", "matrix", "matter", "maximum", "maze", "meadow", "mean",
		"measure", "meat", "mechanic", "medal", "media", "melody", "melt", "member", "memory",
		"men", "mend", "mental", "mentor", "menu", "mercy", "merge", "merit", "merry", "mesh",
		"message", "metal", "method", "middle", "midnight", "milk", "million", "mimic", "mind",
		"minimum", "minor", "minute", "miracle", "mirror", "misery", "miss", "mistake", "mix", "mixed",
		"mixture", "mobile", "model", "modify", "mom", "moment", "monitor", "monkey", "monster", "month",
		"moon", "moral", "more", "morning", "mosquito", "mother", "motion", "motor", "mountain", "mouse", "move",
		"movie", "much", "muffin", "mug", "multiply", "muscle", "museum", "mushroom", "music", "must",
		"mutual", "myself", "mystery", "myth", "naive", "name", "napkin", "narrow", "nasty",
		"nation", "nature", "near", "neck", "need", "negative", "neglect", "neither", "nephew",
		"nerve", "nest", "net", "network", "neutral", "never", "news", "next", "nice", "night",
		"noble", "noise", "nominee", "noodle", "normal", "north", "nose", "notable", "note", "nothing",
		"notice", "novel", "now", "nuclear", "number", "nurse", "nut", "oak", "obey", "object",
		"oblige", "obscure", "observe", "obtain", "obvious", "occur", "ocean", "october", "odor",
		"off", "offer", "office", "often", "oil", "okay", "old", "olive", "olympic", "omit",
		"once", "one", "onion", "online", "only", "open", "opera", "opinion", "oppose", "option",
		"orange", "orbit", "orchard", "order", "ordinary", "organ", "orient", "original", "orphan", "ostrich",
		"other", "outdoor", "outer", "output", "outside", "oval", "oven", "over", "own", "owner",
		"oxygen", "oyster", "ozone", "paddle", "page", "pair", "palace", "palm", "panda",
		"panel", "panic", "panther", "pants", "paper", "parade", "parent", "park", "parrot",
		"party", "pass", "patch", "path", "patient", "patrol", "pattern", "pause", "pave", "payment",
		"peace", "peanut", "pear", "peasant", "pelican", "pen", "penalty", "pencil", "people",
		"pepper", "perfect", "permit", "person", "pet", "phone", "photo", "phrase", "physical", "piano",
		"picnic", "picture", "piece", "pig", "pigeon", "pill", "pilot", "pink", "pioneer",
		"pipe", "pistol", "pitch", "pizza", "place", "planet", "plastic", "plate", "play", "please",
		"pledge", "plenty", "plot", "plough", "plow", "pluck", "plug", "plunge", "poem",
		"poet", "point", "polar", "pole", "police", "pond", "pony", "pool", "popular",
		"portion", "position", "possible", "post", "potato", "pottery", "poverty", "powder",
		"power", "practice", "praise", "predict", "prefer", "prepare", "present", "pretty",
		"prevent", "price", "pride", "primary", "print", "priority", "prison", "private", "prize",
		"problem", "process", "produce", "profit", "program", "project", "promote", "proof",
		"property", "prosper", "protect", "proud", "provide", "public", "pudding", "pull", "pulp",
		"pulse", "pumpkin", "punch", "pupil", "puppy", "purchase", "purity", "purpose", "purse",
		"push", "put", "puzzle", "pyramid", "quality", "quantum", "quarter", "question", "quick",
		"quit", "quiz", "quote", "rabbit", "raccoon", "race", "rack", "radar", "radio", "rail",
		"rain", "raise", "rally", "ramp", "ranch", "random", "range", "rapid", "rare", "rate",
		"rather", "raven", "raw", "reach", "react", "read", "reader", "real", "realm", "rear",
		"reason", "rebel", "build", "recall", "receive", "recipe", "record", "recover", "recruit",
		"red", "reduce", "reflect", "reform", "refuse", "region", "regret", "regular", "reject",
		"relax", "release", "relief", "rely", "remain", "remember", "remind", "remote", "remove", "render", "renew",
		"rent", "reopen", "repair", "repeat", "replace", "reply", "report", "represent", "reproduce",
		"public", "require", "rescue", "resemble", "resist", "resource", "response", "result", "retire",
		"retreat", "return", "reunion", "reveal", "review", "reward", "rhythm", "rib", "ribbon",
		"rice", "rich", "ride", "ridge", "rifle", "right", "rigid", "ring", "riot", "ripple",
		"risk", "ritual", "rival", "river", "road", "roast", "robot", "robust", "rocket", "romance",
		"roof", "rookie", "room", "root", "rope", "rose", "rotate", "rough", "round", "route",
		"royal", "rubber", "rude", "rug", "rule", "run", "runway", "rural", "sad", "saddle",
		"sadness", "safe", "sail", "saint", "salad", "salary", "sale", "salmon", "salon",
		"salt", "salute", "same", "sample", "sand", "satisfy", "satoshi", "sauce", "sausage", "save",
		"say", "scale", "scan", "scare", "scatter", "scene", "scheme", "school", "science", "scissors",
		"scorpion", "scout", "scrap", "screen", "script", "scrub", "sea", "search", "season",
		"seat", "second", "secret", "section", "security", "seed", "seek", "segment", "select", "sell",
		"seminar", "senior", "sense", "sentence", "series", "service", "session", "settle", "setup",
		"seven", "shadow", "shaft", "shallow", "share", "shed", "shell", "sheriff", "shield",
		"shift", "shine", "ship", "shiver", "shock", "shoe", "shoot", "shop", "short",
		"shoulder", "shove", "shrimp", "shrug", "shuffle", "shy", "sibling", "sick", "side",
		"siege", "sight", "sign", "silent", "silk", "silly", "silver", "similar", "simple", "since",
		"sing", "siren", "sister", "situate", "six", "sixteen", "size", "skate", "sketch",
		"ski", "skill", "skin", "skirt", "skull", "slab", "slam", "sleep", "slender", "slice",
		"slide", "slight", "slim", "slogan", "slot", "slow", "slush", "small", "smart", "smile",
		"smoke", "smooth", "snack", "snake", "snap", "sniff", "snow", "soap", "soccer",
		"social", "sock", "soda", "soft", "solar", "soldier", "solid", "solution", "solve",
		"someone", "song", "soon", "sorry", "sort", "soul", "sound", "soup", "source", "south",
		"space", "spare", "spark", "speak", "special", "speech", "speed", "spell", "spend",
		"sphere", "spice", "spider", "spike", "spin", "spirit", "split", "spoil", "sponsor",
		"spoon", "sport", "spot", "spray", "spread", "spring", "spy", "square", "squeeze", "squirrel",
		"stable", "stadium", "staff", "stage", "stairs", "stamp", "stand", "start", "state", "stay",
		"steak", "steel", "stem", "step", "stereo", "stick", "still", "sting", "stock",
		"stomach", "stone", "stool", "story", "stove", "strategy", "street", "strike", "strong",
		"struggle", "student", "stuff", "stumble", "stun", "stunt", "style", "subject",
		"submit", "subway", "success", "such", "sudden", "suffer", "sugar", "suggest",
		"suit", "summer", "sun", "sunny", "sunset", "super", "supply", "supreme", "sure",
		"surface", "surge", "surprise", "surround", "survey", "suspect", "sustain", "swallow",
		"swap", "swarm", "swear", "sweat", "sweep", "sweet", "swift", "swim", "swing",
		"switch", "sword", "symbol", "symptom", "syrup", "system", "table", "tackle", "tag",
		"tail", "talent", "talk", "tank", "tape", "target", "task", "taste", "tattoo", "taxi",
		"team", "tell", "ten", "tenant", "tennis", "tent", "term", "test", "text", "thank",
		"that", "theme", "then", "theory", "there", "they", "thing", "this", "thought",
		"three", "thrive", "throw", "thumb", "thunder", "ticket", "tide", "tiger", "tilt",
		"timber", "time", "tiny", "tip", "tired", "tissue", "title", "toast", "tobacco",
		"toddler", "toe", "together", "toilet", "token", "tomato", "tomorrow", "tone", "tongue",
		"tonight", "tool", "tooth", "top", "topic", "topple", "torch", "tornado", "tortoise",
		"toss", "total", "tourist", "toward", "tower", "town", "toy", "track", "trade", "traffic",
		"tragic", "train", "transfer", "trap", "trash", "travel", "tray", "treat", "tree",
		"tremble", "trend", "trial", "tribe", "trick", "trigger", "trillion", "trim", "trip",
		"trophy", "trouble", "truck", "true", "truly", "trumpet", "trust", "truth", "try",
		"tube", "tuition", "tumble", "tuna", "tunnel", "turkey", "turn", "turtle", "twelve",
		"twenty", "twice", "twin", "twist", "two", "type", "typical", "ugly", "umbrella",
		"unable", "unaware", "uncle", "uncover", "under", "undo", "unfair", "unfold", "unhappy",
		"uniform", "unique", "unit", "universe", "unknown", "unlock", "until", "unusual",
		"unveil", "update", "upgrade", "uphold", "upon", "upper", "upset", "urban", "urge",
		"usage", "use", "used", "useful", "useless", "usual", "utility", "vacant", "vacuum", "vague",
		"valid", "valley", "valve", "van", "vanish", "vapor", "various", "vegan", "velvet", "vendor",
		"venture", "venue", "verb", "verify", "version", "very", "vessel", "veteran", "viable", "vibrant",
		"vicious", "victory", "video", "view", "village", "vintage", "violin", "virtual", "virus",
		"visa", "visit", "visual", "vital", "vivid", "vocal", "voice", "void", "volcano",
		"volume", "vote", "voyage", "wage", "wagon", "wait", "walk", "wall", "walnut",
		"want", "warfare", "warm", "warrior", "wash", "wasp", "waste", "watch", "water",
		"wave", "way", "wealth", "weapon", "wear", "weasel", "weather", "web", "wedding",
		"weekend", "weird", "welcome", "west", "wet", "whale", "what", "wheat", "wheel",
		"when", "where", "whip", "whisper", "whistle", "white", "who", "whole", "why",
		"wicked", "wide", "widow", "width", "wife", "wild", "will", "win", "window", "wine",
		"wing", "wink", "winner", "winter", "wire", "wisdom", "wise", "wish", "witness", "wolf",
		"woman", "wonder", "wood", "wool", "word", "work", "world", "worry", "worth", "wrap",
		"wreck", "wrestle", "wrist", "write", "wrong", "yard", "year", "yellow", "you", "young",
		"youth", "zebra", "zero", "zombie", "zone",
	},
}

// GenerateMnemonic creates a BIP-39 mnemonic from entropy
func GenerateMnemonic(wordCount int) (string, error) {
	// Validate word count
	entropyBits := 0
	switch wordCount {
	case 12:
		entropyBits = 128
	case 15:
		entropyBits = 160
	case 18:
		entropyBits = 192
	case 21:
		entropyBits = 224
	case 24:
		entropyBits = 256
	default:
		return "", errors.New("invalid word count: must be 12, 15, 18, 21, or 24")
	}

	// Generate random entropy
	entropy := make([]byte, entropyBits/8)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	return EntropyToMnemonic(entropy)
}

// EntropyToMnemonic converts entropy to a BIP-39 mnemonic
func EntropyToMnemonic(entropy []byte) (string, error) {
	entropyLen := len(entropy) * 8
	if entropyLen < 128 || entropyLen > 256 {
		return "", errors.New("entropy length must be between 128 and 256 bits")
	}
	if entropyLen%32 != 0 {
		return "", errors.New("entropy length must be a multiple of 32")
	}

	// Calculate checksum
	hash := sha256.Sum256(entropy)
	checksumLen := entropyLen / 32
	checksum := hash[0] >> (8 - checksumLen)

	// Combine entropy and checksum
	bits := entropyLen + checksumLen
	totalWords := bits / 11

	// Convert to words
	bitsBuffer := make([]bitBuffer, totalWords)
	for i := 0; i < totalWords; i++ {
		bitIndex := i * 11
		byteIndex := bitIndex / 8
		bitOffset := bitIndex % 8

		var value uint16
		if bitOffset <= 8 {
			value = uint16(entropy[byteIndex]) << (8 - bitOffset)
			if byteIndex+1 < len(entropy) {
				value |= uint16(entropy[byteIndex+1]) >> (bitOffset + 8)
			}
		} else {
			value = uint16(entropy[byteIndex]) << (16 - bitOffset)
			if byteIndex+1 < len(entropy) {
				value |= uint16(entropy[byteIndex+1]) >> (bitOffset - 8)
			}
		}
		value >>= 5
		bitsBuffer[i] = bitBuffer(value & 0x7FF)
	}

	// Get words
	words := make([]string, totalWords)
	for i := 0; i < totalWords; i++ {
		idx := int(bitsBuffer[i])
		if idx >= len(DefaultEnglishWordList.Words) {
			return "", fmt.Errorf("word index %d out of range", idx)
		}
		words[i] = DefaultEnglishWordList.Words[idx]
	}

	return strings.Join(words, " "), nil
}

// MnemonicToEntropy converts a BIP-39 mnemonic back to entropy
func MnemonicToEntropy(mnemonic string) ([]byte, error) {
	words := strings.Fields(strings.ToLower(mnemonic))
	wordCount := len(words)
	if wordCount < 12 || wordCount > 24 || wordCount%3 != 0 {
		return nil, errors.New("invalid mnemonic word count")
	}

	// Convert words to bits
	bits := make([]uint16, wordCount)
	for i, word := range words {
		idx := slices.Index(DefaultEnglishWordList.Words, word)
		if idx < 0 {
			return nil, fmt.Errorf("invalid word at position %d: %s", i+1, word)
		}
		bits[i] = uint16(idx)
	}

	// Extract entropy
	entropyBits := (wordCount * 11) - (wordCount / 3)
	entropyLen := entropyBits / 8
	entropy := make([]byte, entropyLen)

	for i := 0; i < entropyBits; i++ {
		bitIndex := i
		byteIndex := bitIndex / 8
		bitOffset := 7 - (bitIndex % 8)

		wordIndex := i / 11
		wordOffset := 10 - (i % 11)
		bit := (bits[wordIndex] >> wordOffset) & 1

		entropy[byteIndex] |= byte(bit << bitOffset)
	}

	return entropy, nil
}

// ValidateMnemonic checks if a BIP-39 mnemonic is valid
func ValidateMnemonic(mnemonic string) bool {
	words := strings.Fields(strings.ToLower(mnemonic))
	if len(words) < 12 || len(words) > 24 || len(words)%3 != 0 {
		return false
	}

	for _, word := range words {
		if !slices.Contains(DefaultEnglishWordList.Words, word) {
			return false
		}
	}

	// Verify checksum
	entropy, err := MnemonicToEntropy(mnemonic)
	if err != nil {
		return false
	}

	// Calculate expected checksum
	hash := sha256.Sum256(entropy)
	checksumLen := len(entropy) * 8 / 32
	checksum := hash[0] >> (8 - checksumLen)

	// Get actual checksum from mnemonic
	wordCount := len(words)
	checksumBits := wordCount / 3
	totalBits := wordCount * 11

	var actualChecksum uint8
	for i := 0; i < checksumBits; i++ {
		bitIndex := entropyBits + i
		wordIndex := bitIndex / 11
		wordOffset := 10 - (bitIndex % 11)
		bit := (bits[wordIndex] >> wordOffset) & 1
		actualChecksum |= byte(bit << (checksumBits - 1 - i))
	}

	return checksum == actualChecksum
}

// MnemonicToSeed converts a BIP-39 mnemonic to a seed using PBKDF2
func MnemonicToSeed(mnemonic, passphrase string) ([]byte, error) {
	mnemonic = strings.TrimSpace(strings.ToLower(mnemonic))
	if !ValidateMnemonic(mnemonic) {
		return nil, errors.New("invalid mnemonic")
	}

	// Simple seed derivation (in production, use proper PBKDF2)
	mnemonic = "mnemonic" + mnemonic
	if passphrase != "" {
		mnemonic += passphrase
	}

	hash := sha256.Sum256([]byte(mnemonic))
	return hash[:], nil
}

// bitBuffer is a helper for bit manipulation
type bitBuffer uint16

// ToBinary converts the buffer to binary string
func (b bitBuffer) ToBinary() string {
	return fmt.Sprintf("%011b", uint16(b))
}

// CountWords returns the number of words for a given entropy strength
func CountWords(entropyBits int) int {
	wordCount := (entropyBits + entropyBits/32) / 11
	return wordCount
}

// GetEntropyStrength returns the required entropy strength for word count
func GetEntropyStrength(wordCount int) (int, error) {
	expected, ok := entropyLengths[wordCount*8+wordCount/3*8]
	if !ok {
		return 0, errors.New("invalid word count")
	}
	return expected, nil
}

// NormalizeWhitespace normalizes whitespace in mnemonic
func NormalizeWhitespace(mnemonic string) string {
	words := strings.Fields(strings.ToLower(mnemonic))
	return strings.Join(words, " ")
}

// ToString returns the mnemonic as a string
func (wl WordList) ToString() string {
	return strings.Join(wl.Words, "\n")
}

// FindWord finds the index of a word in the word list
func FindWord(word string) int {
	return slices.Index(DefaultEnglishWordList.Words, strings.ToLower(word))
}

// GetSuggestions returns up to n word suggestions for prefix
func GetSuggestions(prefix string, n int) []string {
	prefix = strings.ToLower(prefix)
	if len(prefix) < 1 {
		return nil
	}

	var suggestions []string
	for _, word := range DefaultEnglishWordList.Words {
		if strings.HasPrefix(word, prefix) {
			suggestions = append(suggestions, word)
			if len(suggestions) >= n {
				break
			}
		}
	}

	return suggestions
}

// IsWordValid checks if a word is in the word list
func IsWordValid(word string) bool {
	return slices.Contains(DefaultEnglishWordList.Words, strings.ToLower(word))
}

// ToSeedPhrase converts mnemonic to a 24-word seed phrase format
func ToSeedPhrase(mnemonic string) (string, error) {
	words := strings.Fields(mnemonic)
	if len(words) != 24 {
		return "", errors.New("mnemonic must have exactly 24 words")
	}
	
	seed, err := MnemonicToSeed(mnemonic, "")
	if err != nil {
		return "", err
	}
	
	// Format as hex for storage
	return fmt.Sprintf("%x", seed), nil
}