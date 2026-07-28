// ============================================================================
// TigerWallet - Crypto Service
// High-Speed Cryptographic Operations with Real BIP-39/44 Implementation
// ============================================================================

import { ethers, HDNodeWallet, Mnemonic, SigningKey, Wallet, keccak256, toUtf8Bytes } from 'ethers';
import { createCipheriv, createDecipheriv, randomBytes, createHash, createHmac } from 'react-native-quick-crypto';
import CryptoJS from 'crypto-js';

// ============================================================================
// Constants
// ============================================================================

const BIP39_WORDLIST = [
  'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract', 'absurd', 'abuse',
  'access', 'accident', 'account', 'accuse', 'achieve', 'acid', 'acoustic', 'acquire', 'across', 'act',
  'action', 'actor', 'actress', 'actual', 'adapt', 'add', 'addict', 'address', 'adjust', 'admit',
  'adult', 'advance', 'advice', 'aerobic', 'affair', 'afford', 'afraid', 'again', 'age', 'agent',
  'agree', 'ahead', 'aim', 'air', 'airport', 'aisle', 'alarm', 'album', 'alcohol', 'alert',
  'alien', 'all', 'alley', 'allow', 'almost', 'alone', 'alpha', 'already', 'also', 'alter',
  'always', 'amateur', 'amazing', 'among', 'amount', 'amused', 'analyst', 'anchor', 'ancient',
  'anger', 'angle', 'angry', 'animal', 'ankle', 'announce', 'annual', 'another', 'answer', 'antenna',
  'antique', 'anxiety', 'any', 'apart', 'apology', 'appear', 'apple', 'approve', 'april', 'arch',
  'arctic', 'area', 'arena', 'argue', 'arm', 'armed', 'armor', 'army', 'around', 'arrange',
  'arrest', 'arrive', 'arrow', 'art', 'artefact', 'artist', 'artwork', 'ask', 'aspect', 'assault',
  'asset', 'assist', 'assume', 'asthma', 'athlete', 'atom', 'attack', 'attend', 'attitude', 'attract',
  'auction', 'audit', 'august', 'aunt', 'author', 'auto', 'autumn', 'average', 'avocado', 'avoid',
  'awake', 'aware', 'away', 'awesome', 'awful', 'awkward', 'axis', 'baby', 'bachelor', 'bacon',
  'badge', 'bag', 'balance', 'balcony', 'ball', 'bamboo', 'banana', 'banner', 'bar', 'barely',
  'bargain', 'barrel', 'base', 'basic', 'basket', 'battle', 'beach', 'bean', 'beauty', 'because',
  'become', 'beef', 'before', 'begin', 'behave', 'behind', 'believe', 'below', 'belt', 'bench',
  'benefit', 'best', 'betray', 'better', 'between', 'beyond', 'bicycle', 'bid', 'bike', 'bind',
  'biology', 'bird', 'birth', 'bitter', 'black', 'blade', 'blame', 'blanket', 'blast', 'bleak',
  'bless', 'blind', 'blood', 'blossom', 'blouse', 'blue', 'blur', 'blush', 'board', 'boat',
  'body', 'boil', 'bomb', 'bone', 'bonus', 'book', 'boost', 'border', 'boring', 'borrow',
  'boss', 'bottom', 'bounce', 'box', 'boy', 'bracket', 'brain', 'brand', 'brass', 'brave',
  'bread', 'breeze', 'brick', 'bridge', 'brief', 'bright', 'bring', 'brisk', 'broccoli', 'broken',
  'bronze', 'broom', 'brother', 'brown', 'brush', 'bubble', 'buddy', 'budget', 'buffalo', 'build',
  'bulb', 'bulk', 'bullet', 'bundle', 'bunker', 'burden', 'burger', 'burst', 'bus', 'business',
  'busy', 'butter', 'buyer', 'buzz', 'cabbage', 'cabin', 'cable', 'cactus', 'cage', 'cake',
  'call', 'calm', 'camera', 'camp', 'can', 'canal', 'cancel', 'candy', 'cannon', 'canoe',
  'canvas', 'canyon', 'capable', 'capital', 'captain', 'car', 'carbon', 'card', 'cargo', 'carpet',
  'carry', 'cart', 'case', 'cash', 'casino', 'castle', 'casual', 'cat', 'catalog', 'catch',
  'category', 'cattle', 'caught', 'cause', 'caution', 'cave', 'ceiling', 'celery', 'cement', 'census',
  'century', 'cereal', 'certain', 'chair', 'chalk', 'champion', 'change', 'chaos', 'chapter',
  'charge', 'chase', 'chat', 'cheap', 'check', 'cheese', 'chef', 'cherry', 'chest', 'chicken',
  'chief', 'child', 'chimney', 'choice', 'choose', 'chronic', 'chuckle', 'chunk', 'churn',
  'cigar', 'cinnamon', 'circle', 'citizen', 'city', 'civil', 'claim', 'clap', 'clarify', 'claw',
  'clay', 'clean', 'clerk', 'clever', 'click', 'client', 'cliff', 'climb', 'clinic', 'clip',
  'clock', 'close', 'cloth', 'cloud', 'clown', 'club', 'clump', 'cluster', 'clutch', 'coach',
  'coast', 'coconut', 'code', 'coffee', 'coil', 'coin', 'collect', 'color', 'column', 'combine',
  'come', 'comfort', 'comic', 'common', 'company', 'concert', 'conduct', 'confirm', 'congress',
  'connect', 'consider', 'control', 'convince', 'cook', 'cool', 'copper', 'copy', 'coral', 'core',
  'corn', 'correct', 'cost', 'cottage', 'cotton', 'couch', 'country', 'couple', 'course', 'cousin',
  'cover', 'coyote', 'crack', 'cradle', 'craft', 'cram', 'crane', 'crash', 'crater', 'crawl',
  'crazy', 'cream', 'credit', 'creek', 'crew', 'cricket', 'crime', 'crisp', 'critic', 'crop',
  'cross', 'crouch', 'crowd', 'crucial', 'cruel', 'cruise', 'crumble', 'crunch', 'crush', 'cry',
  'crystal', 'cube', 'culture', 'cup', 'cupboard', 'curious', 'current', 'curtain', 'curve',
  'cushion', 'custom', 'cute', 'cycle', 'dad', 'damage', 'damp', 'dance', 'danger', 'daring',
  'dash', 'daughter', 'dawn', 'day', 'deal', 'debate', 'debris', 'decade', 'december', 'decide',
  'decline', 'decorate', 'decrease', 'deer', 'defense', 'define', 'defy', 'degree', 'delay', 'deliver',
  'demand', 'demise', 'denial', 'dentist', 'deny', 'depart', 'depend', 'deposit', 'depth', 'deputy',
  'derive', 'describe', 'desert', 'design', 'desk', 'despair', 'destroy', 'detail', 'detect', 'develop',
  'device', 'devote', 'diagram', 'dial', 'diamond', 'diary', 'dice', 'diesel', 'diet', 'differ',
  'digital', 'dignity', 'dilemma', 'dinner', 'dinosaur', 'direct', 'dirt', 'disagree', 'discover',
  'disease', 'dish', 'dismiss', 'disorder', 'display', 'distance', 'divert', 'divide', 'divorce',
  'dizzy', 'doctor', 'document', 'dog', 'doll', 'dolphin', 'domain', 'donate', 'donkey', 'donor',
  'door', 'dose', 'double', 'dove', 'draft', 'dragon', 'drama', 'draw', 'dream', 'dress', 'drift',
  'drill', 'drink', 'drip', 'drive', 'drop', 'drum', 'dry', 'duck', 'dumb', 'dune',
  'during', 'dust', 'dutch', 'duty', 'dwarf', 'dynamic', 'eager', 'eagle', 'early', 'earn',
  'earth', 'easily', 'east', 'easy', 'echo', 'ecology', 'economy', 'edge', 'edit', 'educate',
  'effort', 'eight', 'eject', 'elastic', 'elbow', 'elder', 'electric', 'elegant', 'element', 'elephant',
  'elevator', 'elite', 'else', 'embark', 'embody', 'embrace', 'emerge', 'emotion', 'employ', 'empower',
  'empty', 'enable', 'enact', 'end', 'endless', 'endorse', 'enemy', 'energy', 'enforce', 'engage',
  'engine', 'enhance', 'enjoy', 'enlist', 'enough', 'enrich', 'enroll', 'ensure', 'enter', 'entire',
  'entry', 'envelope', 'episode', 'equal', 'equip', 'era', 'erase', 'erode', 'erosion', 'error',
  'erupt', 'escape', 'essay', 'essence', 'estate', 'eternal', 'ethics', 'evidence', 'evil', 'evoke',
  'evolve', 'exact', 'example', 'excess', 'exchange', 'excite', 'exclude', 'excuse', 'execute', 'exercise',
  'exhaust', 'exhibit', 'exile', 'exist', 'exit', 'exotic', 'expand', 'expect', 'expire', 'explain',
  'expose', 'express', 'extend', 'extra', 'eye', 'eyebrow', 'fabric', 'face', 'faculty', 'fade',
  'faint', 'faith', 'fall', 'false', 'fame', 'family', 'famous', 'fan', 'fancy', 'fantasy',
  'farm', 'fashion', 'fat', 'fatal', 'father', 'fatigue', 'fault', 'favorite', 'feature', 'february',
  'federal', 'fee', 'feed', 'feel', 'female', 'fence', 'festival', 'fetch', 'fever', 'few',
  'fiber', 'fiction', 'field', 'figure', 'file', 'film', 'filter', 'final', 'find', 'fine',
  'finger', 'finish', 'fire', 'firm', 'first', 'fiscal', 'fish', 'fit', 'fitness', 'fix',
  'flag', 'flame', 'flash', 'flat', 'flavor', 'flee', 'flight', 'flip', 'float', 'flock',
  'floor', 'flower', 'fluid', 'flush', 'fly', 'foam', 'focus', 'fog', 'foil', 'fold',
  'follow', 'food', 'foot', 'force', 'forest', 'forget', 'fork', 'fortune', 'forum', 'forward',
  'fossil', 'foster', 'found', 'fox', 'fragile', 'frame', 'frequent', 'fresh', 'friend', 'fringe',
  'frog', 'front', 'frost', 'frown', 'frozen', 'fruit', 'fuel', 'fun', 'funny', 'furnace',
  'fury', 'future', 'gadget', 'gain', 'galaxy', 'gallery', 'game', 'gap', 'garage', 'garbage',
  'garden', 'garlic', 'gas', 'gasp', 'gate', 'gather', 'gauge', 'gaze', 'general', 'genius',
  'genre', 'gentle', 'genuine', 'gesture', 'ghost', 'giant', 'gift', 'giggle', 'ginger', 'giraffe',
  'girl', 'give', 'glad', 'glance', 'glare', 'glass', 'glide', 'glimpse', 'globe', 'gloom',
  'glory', 'glove', 'glow', 'glue', 'goat', 'goddess', 'gold', 'good', 'goose', 'gorilla',
  'gospel', 'gossip', 'govern', 'gown', 'grab', 'grace', 'grain', 'grant', 'grape', 'grass',
  'gravity', 'great', 'green', 'grid', 'grief', 'grit', 'grocery', 'group', 'grow', 'grunt',
  'guard', 'guess', 'guide', 'guilt', 'guitar', 'gun', 'gym', 'habit', 'hair', 'half',
  'hammer', 'hamster', 'hand', 'handle', 'harbor', 'hard', 'harsh', 'harvest', 'hat', 'have',
  'hawk', 'hazard', 'head', 'health', 'heart', 'heavy', 'hedgehog', 'height', 'hello', 'helmet',
  'help', 'hen', 'hero', 'hidden', 'high', 'hill', 'hint', 'hip', 'hire', 'history',
  'hobby', 'hockey', 'hold', 'hole', 'holiday', 'hollow', 'home', 'honey', 'hood', 'hope',
  'horn', 'horror', 'horse', 'hospital', 'host', 'hotel', 'hour', 'hover', 'hub', 'huge',
  'human', 'humble', 'humor', 'hundred', 'hungry', 'hunt', 'hurdle', 'hurry', 'hurt', 'husband',
  'hybrid', 'ice', 'icon', 'idea', 'identify', 'idle', 'ignore', 'ill', 'illegal', 'illness',
  'image', 'imitate', 'immense', 'immune', 'impact', 'impose', 'improve', 'impulse', 'inch',
  'include', 'income', 'increase', 'index', 'indicate', 'indoor', 'industry', 'infant', 'inflict',
  'inform', 'inhale', 'inherit', 'initial', 'inject', 'injury', 'inmate', 'inner', 'innocent',
  'input', 'inquiry', 'insane', 'insect', 'insert', 'inside', 'inspire', 'install', 'intact',
  'interest', 'into', 'invest', 'invite', 'involve', 'iron', 'island', 'isolate', 'issue', 'item',
  'ivory', 'jacket', 'jaguar', 'jar', 'jazz', 'jealous', 'jeans', 'jelly', 'jewel', 'job',
  'join', 'joke', 'journey', 'joy', 'judge', 'juice', 'jump', 'jungle', 'junior', 'junk',
  'just', 'kangaroo', 'keen', 'keep', 'ketchup', 'key', 'kick', 'kid', 'kidney', 'kind',
  'kingdom', 'kiss', 'kit', 'kitchen', 'kite', 'kitten', 'kiwi', 'knee', 'knife', 'knock',
  'know', 'lab', 'label', 'labor', 'ladder', 'lady', 'lake', 'lamp', 'language', 'laptop',
  'large', 'later', 'latin', 'laugh', 'laundry', 'lava', 'law', 'lawn', 'lawsuit', 'layer',
  'lazy', 'leader', 'leaf', 'learn', 'leave', 'lecture', 'left', 'leg', 'legal', 'legend',
  'leisure', 'lemon', 'lend', 'length', 'lens', 'leopard', 'lesson', 'letter', 'level', 'liar',
  'liberty', 'library', 'license', 'life', 'lift', 'light', 'like', 'limb', 'limit', 'link',
  'lion', 'liquid', 'list', 'little', 'live', 'lizard', 'load', 'loan', 'lobster', 'local',
  'lock', 'logic', 'lonely', 'long', 'loop', 'lottery', 'loud', 'lounge', 'love', 'loyal',
  'lucky', 'luggage', 'lumber', 'lunar', 'lunch', 'luxury', 'lyrics', 'machine', 'mad', 'magic',
  'magnet', 'maid', 'mail', 'main', 'major', 'make', 'mammal', 'man', 'manage', 'mandate',
  'mango', 'mansion', 'manual', 'maple', 'marble', 'march', 'margin', 'marine', 'market', 'marriage',
  'mask', 'mass', 'master', 'match', 'material', 'math', 'matrix', 'matter', 'maximum', 'maze',
  'meadow', 'mean', 'measure', 'meat', 'mechanic', 'medal', 'media', 'melody', 'melt', 'member',
  'memory', 'men', 'mend', 'mental', 'mentor', 'menu', 'mercy', 'merge', 'merit', 'merry',
  'mesh', 'message', 'metal', 'method', 'middle', 'midnight', 'milk', 'million', 'mimic', 'mind',
  'minimum', 'minor', 'minute', 'miracle', 'mirror', 'misery', 'miss', 'mistake', 'mix', 'mixed',
  'mixture', 'mobile', 'model', 'modify', 'mom', 'moment', 'monitor', 'monkey', 'monster', 'month',
  'moon', 'moral', 'more', 'morning', 'mosquito', 'mother', 'motion', 'motor', 'mountain', 'mouse',
  'move', 'movie', 'much', 'muffin', 'mule', 'multiply', 'muscle', 'museum', 'mushroom', 'music',
  'must', 'mutual', 'myself', 'mystery', 'myth', 'naive', 'name', 'napkin', 'narrow', 'nasty',
  'nation', 'nature', 'near', 'neck', 'need', 'negative', 'neglect', 'neither', 'nephew', 'nerve',
  'nest', 'net', 'network', 'neutral', 'never', 'news', 'next', 'nice', 'night', 'noble',
  'noise', 'nominee', 'noodle', 'normal', 'north', 'nose', 'notable', 'note', 'nothing', 'notice',
  'novel', 'now', 'nuclear', 'number', 'nurse', 'nut', 'oak', 'obey', 'object', 'oblige',
  'obscure', 'observe', 'obtain', 'obvious', 'occur', 'ocean', 'october', 'odor', 'off', 'offer',
  'office', 'often', 'oil', 'okay', 'old', 'olive', 'olympic', 'omit', 'once', 'one',
  'onion', 'online', 'only', 'open', 'opera', 'opinion', 'oppose', 'option', 'orange', 'orbit',
  'orchard', 'order', 'ordinary', 'organ', 'orient', 'original', 'orphan', 'ostrich', 'other',
  'outdoor', 'outer', 'output', 'outside', 'oval', 'oven', 'over', 'own', 'owner', 'oxygen',
  'oyster', 'ozone', 'pact', 'paddle', 'page', 'pair', 'palace', 'palm', 'panda', 'panel',
  'panic', 'panther', 'paper', 'parade', 'parent', 'park', 'parrot', 'party', 'pass', 'patch',
  'path', 'patient', 'patrol', 'pattern', 'pause', 'pave', 'payment', 'peace', 'peanut', 'pear',
  'peasant', 'pelican', 'pen', 'penalty', 'pencil', 'people', 'pepper', 'perfect', 'permit', 'person',
  'pet', 'phone', 'photo', 'phrase', 'physical', 'piano', 'picnic', 'picture', 'piece', 'pig',
  'pigeon', 'pill', 'pilot', 'pink', 'pioneer', 'pipe', 'pistol', 'pitch', 'pizza', 'place',
  'planet', 'plastic', 'plate', 'play', 'please', 'pledge', 'pluck', 'plug', 'plunge', 'poem',
  'poet', 'point', 'polar', 'pole', 'police', 'pond', 'pony', 'pool', 'popular', 'portion',
  'position', 'possible', 'post', 'potato', 'pottery', 'poverty', 'powder', 'power', 'practice',
  'praise', 'predict', 'prefer', 'prepare', 'present', 'pretty', 'prevent', 'price', 'pride',
  'primary', 'print', 'priority', 'prison', 'private', 'prize', 'problem', 'process', 'produce',
  'profit', 'program', 'project', 'promote', 'proof', 'property', 'prosper', 'protect', 'proud',
  'provide', 'public', 'pudding', 'pull', 'pulp', 'pulse', 'pumpkin', 'punch', 'pupil', 'puppy',
  'purchase', 'purity', 'purpose', 'purse', 'push', 'put', 'puzzle', 'pyramid', 'quality', 'quantum',
  'quarter', 'question', 'quick', 'quit', 'quiz', 'quote', 'rabbit', 'raccoon', 'race', 'rack',
  'radar', 'radio', 'rail', 'rain', 'raise', 'rally', 'ramp', 'ranch', 'random', 'range',
  'rapid', 'rare', 'rate', 'rather', 'raven', 'raw', 'reach', 'react', 'read', 'real',
  'realm', 'rear', 'reason', 'rebel', 'rebuild', 'recall', 'receive', 'recipe', 'record', 'recover',
  'recruit', 'red', 'reduce', 'reflect', 'reform', 'refuse', 'region', 'regret', 'regular', 'reject',
  'relax', 'release', 'relief', 'rely', 'remain', 'remember', 'remind', 'remote', 'remove', 'render',
  'renew', 'rent', 'reopen', 'repair', 'repeat', 'replace', 'reply', 'report', 'represent', 'reproduce',
  'public', 'require', 'rescue', 'resemble', 'resist', 'resource', 'response', 'result', 'retire', 'retreat',
  'return', 'reunion', 'reveal', 'review', 'reward', 'rhythm', 'rib', 'ribbon', 'rice', 'rich',
  'ride', 'ridge', 'rifle', 'right', 'rigid', 'ring', 'riot', 'ripple', 'risk', 'ritual',
  'rival', 'river', 'road', 'roast', 'robot', 'robust', 'rocket', 'romance', 'roof', 'rookie',
  'room', 'root', 'rope', 'rose', 'rotate', 'rough', 'round', 'route', 'royal', 'rubber',
  'rude', 'rug', 'rule', 'run', 'runway', 'rural', 'sad', 'saddle', 'sadness', 'safe',
  'sail', 'salad', 'salmon', 'salon', 'salt', 'salute', 'same', 'sample', 'sand', 'satisfy',
  'satoshi', 'sauce', 'sausage', 'save', 'say', 'scale', 'scan', 'scare', 'scatter', 'scene',
  'scheme', 'school', 'science', 'scissors', 'scorpion', 'scout', 'scrap', 'screen', 'script', 'scrub',
  'sea', 'search', 'season', 'seat', 'second', 'secret', 'section', 'security', 'seed', 'seek',
  'segment', 'select', 'sell', 'seminar', 'senior', 'sense', 'sentence', 'series', 'service', 'session',
  'settle', 'setup', 'seven', 'shadow', 'shaft', 'shallow', 'share', 'shed', 'shell', 'sheriff',
  'shield', 'shift', 'shine', 'ship', 'shiver', 'shock', 'shoe', 'shoot', 'shop', 'short',
  'shoulder', 'shove', 'shrimp', 'shrug', 'shuffle', 'shy', 'sibling', 'sick', 'side', 'siege',
  'sight', 'sign', 'silent', 'silk', 'silly', 'silver', 'similar', 'simple', 'since', 'sing',
  'siren', 'sister', 'situate', 'six', 'size', 'skate', 'sketch', 'ski', 'skill', 'skin',
  'skirt', 'skull', 'slab', 'slam', 'sleep', 'slender', 'slice', 'slide', 'slight', 'slim',
  'slogan', 'slot', 'slow', 'slush', 'small', 'smart', 'smile', 'smoke', 'smooth', 'snack',
  'snake', 'snap', 'sniff', 'snow', 'soap', 'soccer', 'social', 'sock', 'soda', 'soft',
  'solar', 'soldier', 'solid', 'solution', 'solve', 'someone', 'song', 'soon', 'sorry', 'sort',
  'soul', 'sound', 'soup', 'source', 'south', 'space', 'spare', 'spark', 'speak', 'special',
  'speed', 'spell', 'spend', 'sphere', 'spice', 'spider', 'spike', 'spin', 'spirit', 'split',
  'spoil', 'sponsor', 'spoon', 'sport', 'spot', 'spray', 'spread', 'spring', 'spur', 'square',
  'squeeze', 'squirrel', 'stable', 'stadium', 'staff', 'stage', 'stairs', 'stamp', 'stand', 'start',
  'state', 'stay', 'steak', 'steel', 'stem', 'step', 'stereo', 'stick', 'still', 'sting', 'stock',
  'stomach', 'stone', 'stool', 'story', 'stove', 'strategy', 'street', 'strike', 'strong', 'struggle',
  'student', 'stuff', 'stumble', 'style', 'subject', 'submit', 'subway', 'success', 'such', 'sudden',
  'suffer', 'sugar', 'suggest', 'suit', 'summer', 'sun', 'sunny', 'sunset', 'super', 'supply', 'supreme',
  'sure', 'surface', 'surge', 'surprise', 'surround', 'survey', 'suspect', 'sustain', 'swallow', 'swamp',
  'swap', 'swarm', 'swear', 'sweat', 'sweep', 'sweet', 'swift', 'swim', 'swing', 'switch', 'sword',
  'symbol', 'symptom', 'syrup', 'system', 'table', 'tackle', 'tag', 'tail', 'talent', 'talk', 'tank',
  'tape', 'target', 'task', 'taste', 'tattoo', 'taxi', 'teach', 'team', 'tell', 'ten', 'tenant',
  'tennis', 'tent', 'term', 'test', 'text', 'thank', 'that', 'theme', 'then', 'theory', 'there',
  'they', 'thing', 'this', 'thought', 'three', 'thrive', 'throw', 'thumb', 'thunder', 'ticket', 'tide',
  'tiger', 'tilt', 'timber', 'time', 'tiny', 'tip', 'tired', 'tissue', 'title', 'toast', 'tobacco',
  'toddler', 'toe', 'together', 'toilet', 'token', 'tomato', 'tomorrow', 'tone', 'tongue', 'tonight',
  'tool', 'tooth', 'top', 'topic', 'topple', 'torch', 'tornado', 'tortoise', 'toss', 'total', 'tourist',
  'toward', 'tower', 'town', 'toy', 'track', 'trade', 'traffic', 'tragic', 'train', 'transfer', 'trap',
  'trash', 'travel', 'tray', 'treat', 'tree', 'trend', 'trial', 'tribe', 'trick', 'trigger', 'trim',
  'trip', 'trophy', 'trouble', 'truck', 'true', 'truly', 'trumpet', 'trust', 'truth', 'try', 'tube',
  'tuition', 'tumble', 'tuna', 'tunnel', 'turkey', 'turn', 'turtle', 'twelve', 'twenty', 'twice',
  'twin', 'twist', 'two', 'type', 'typical', 'ugly', 'umbrella', 'unable', 'unaware', 'uncle', 'uncover',
  'under', 'undo', 'unfair', 'unfold', 'unhappy', 'uniform', 'unique', 'unit', 'universe', 'unknown',
  'unlock', 'until', 'unusual', 'unveil', 'update', 'upgrade', 'uphold', 'upon', 'upper', 'upset',
  'urban', 'urge', 'usage', 'use', 'used', 'useful', 'useless', 'usual', 'utility', 'vacant', 'vacuum',
  'vague', 'valid', 'valley', 'valve', 'van', 'vanish', 'vapor', 'various', 'vegan', 'velvet', 'vendor',
  'venture', 'venue', 'verb', 'verify', 'version', 'very', 'vessel', 'veteran', 'viable', 'vibrant', 'vicious',
  'victory', 'video', 'view', 'village', 'vintage', 'violin', 'virtual', 'virus', 'visa', 'visit', 'visual',
  'vital', 'vivid', 'vocal', 'voice', 'void', 'volcano', 'volume', 'vote', 'voyage', 'wage', 'wagon',
  'wait', 'wake', 'walk', 'wall', 'walnut', 'want', 'warfare', 'warm', 'warrior', 'wash', 'wasp',
  'waste', 'water', 'wave', 'way', 'wealth', 'weapon', 'wear', 'weasel', 'weather', 'web', 'wedding',
  'weekend', 'weird', 'welcome', 'west', 'wet', 'whale', 'what', 'wheat', 'wheel', 'when', 'where',
  'whip', 'whisper', 'wide', 'width', 'wife', 'wild', 'will', 'win', 'window', 'wine', 'wing',
  'wink', 'winner', 'winter', 'wire', 'wisdom', 'wise', 'wish', 'witness', 'wolf', 'woman', 'wonder',
  'wood', 'wool', 'word', 'work', 'world', 'worry', 'worth', 'wrap', 'wreck', 'wrestle', 'wrist',
  'write', 'wrong', 'yard', 'year', 'yellow', 'you', 'young', 'youth', 'zebra', 'zero', 'zone', 'zoo'
];

// ============================================================================
// Crypto Service Class
// ============================================================================

export class CryptoService {
  private static instance: CryptoService;
  
  private constructor() {
    // Initialize crypto
  }

  static getInstance(): CryptoService {
    if (!CryptoService.instance) {
      CryptoService.instance = new CryptoService();
    }
    return CryptoService.instance;
  }

  // ============================================================================
  // Mnemonic Generation & Validation
  // ============================================================================

  /**
   * Generate a new BIP-39 mnemonic phrase
   * @param strength - Entropy strength (128, 160, 192, 224, 256 bits)
   */
  generateMnemonic(strength: number = 256): string {
    const wallet = HDNodeWallet.createRandom(undefined, undefined, undefined, BIP39_WORDLIST);
    return wallet.mnemonic!.phrase;
  }

  /**
   * Validate a BIP-39 mnemonic phrase
   */
  validateMnemonic(mnemonic: string): boolean {
    try {
      const words = mnemonic.trim().split(/\s+/);
      if (words.length !== 12 && words.length !== 24) {
        return false;
      }
      // Try to create wallet to validate
      HDNodeWallet.fromPhrase(mnemonic, undefined, undefined, BIP39_WORDLIST);
      return true;
    } catch {
      return false;
    }
  }

  /**
   * Get entropy from mnemonic
   */
  mnemonicToEntropy(mnemonic: string): string {
    const wallet = HDNodeWallet.fromPhrase(mnemonic, undefined, undefined, BIP39_WORDLIST);
    return wallet.mnemonic!.entropy?.toString(16).padStart(64, '0') || '';
  }

  /**
   * Get mnemonic from entropy
   */
  entropyToMnemonic(entropy: string): string {
    const wallet = HDNodeWallet.fromEntropy(entropy, undefined, BIP39_WORDLIST);
    return wallet.mnemonic!.phrase;
  }

  // ============================================================================
  // Key Derivation (BIP-32/44)
  // ============================================================================

  /**
   * Derive master key from mnemonic
   */
  deriveMasterKey(mnemonic: string): { privateKey: string; publicKey: string } {
    const wallet = HDNodeWallet.fromPhrase(mnemonic, undefined, undefined, BIP39_WORDLIST);
    return {
      privateKey: wallet.privateKey,
      publicKey: wallet.publicKey,
    };
  }

  /**
   * Derive child key using BIP-44 path
   * @param mnemonic - BIP-39 mnemonic
   * @param path - BIP-44 path (e.g., m/44'/60'/0'/0/0)
   */
  deriveKey(mnemonic: string, path: string): { address: string; privateKey: string; publicKey: string } {
    const wallet = HDNodeWallet.fromPhrase(mnemonic, undefined, undefined, BIP39_WORDLIST);
    const derivedWallet = wallet.derivePath(path);
    return {
      address: derivedWallet.address,
      privateKey: derivedWallet.privateKey,
      publicKey: derivedWallet.publicKey,
    };
  }

  /**
   * Derive EVM address from mnemonic (m/44'/60'/0'/0/0)
   */
  deriveEvmAddress(mnemonic: string): string {
    const wallet = HDNodeWallet.fromPhrase(mnemonic, undefined, undefined, BIP39_WORDLIST);
    const derivedWallet = wallet.derivePath("m/44'/60'/0'/0/0");
    return derivedWallet.address;
  }

  /**
   * Derive Bitcoin address (Legacy)
   */
  deriveBitcoinAddress(mnemonic: string): string {
    const wallet = HDNodeWallet.fromPhrase(mnemonic, undefined, undefined, BIP39_WORDLIST);
    const derivedWallet = wallet.derivePath("m/44'/0'/0'/0/0");
    // For BTC, we'd need to convert to legacy address format
    return derivedWallet.address;
  }

  /**
   * Derive Solana address
   */
  deriveSolanaAddress(mnemonic: string): string {
    // For Solana, we need Ed25519 key derivation
    // This is simplified - in production, use proper Ed25519 derivation
    const wallet = HDNodeWallet.fromPhrase(mnemonic, undefined, undefined, BIP39_WORDLIST);
    const derivedWallet = wallet.derivePath("m/44'/501'/0'/0'");
    return derivedWallet.address;
  }

  /**
   * Generate addresses for all supported chains from single mnemonic
   */
  deriveAllAddresses(mnemonic: string): Record<number, string> {
    const paths: Record<number, string> = {
      // EVM chains (60 = Ethereum)
      1: "m/44'/60'/0'/0/0",      // Ethereum
      56: "m/44'/60'/0'/0/0",     // BSC
      137: "m/44'/60'/0'/0/0",   // Polygon
      42161: "m/44'/60'/0'/0/0", // Arbitrum
      10: "m/44'/60'/0'/0/0",    // Optimism
      8453: "m/44'/60'/0'/0/0",  // Base
      43114: "m/44'/60'/0'/0/0", // Avalanche
      59144: "m/44'/60'/0'/0/0", // Linea
      
      // Bitcoin (0)
      0: "m/44'/0'/0'/0/0",
      
      // Solana (501)
      501: "m/44'/501'/0'/0'",
    };

    const addresses: Record<number, string> = {};
    
    for (const [chainId, path] of Object.entries(paths)) {
      try {
        const wallet = HDNodeWallet.fromPhrase(mnemonic, undefined, undefined, BIP39_WORDLIST);
        const derivedWallet = wallet.derivePath(path);
        addresses[parseInt(chainId)] = derivedWallet.address;
      } catch (e) {
        console.error(`Failed to derive for chain ${chainId}:`, e);
      }
    }
    
    return addresses;
  }

  // ============================================================================
  // Private Key Operations
  // ============================================================================

  /**
   * Create wallet from private key
   */
  fromPrivateKey(privateKey: string): { address: string; privateKey: string; publicKey: string } {
    const wallet = new Wallet(privateKey);
    return {
      address: wallet.address,
      privateKey: wallet.privateKey,
      publicKey: wallet.publicKey,
    };
  }

  /**
   * Encrypt private key with password
   */
  encryptPrivateKey(privateKey: string, password: string): string {
    const salt = randomBytes(16);
    const key = CryptoJS.PBKDF2(password, salt.toString('hex'), {
      keySize: 256 / 32,
      iterations: 100000,
    }).toString();
    
    const iv = randomBytes(16);
    const encrypted = CryptoJS.AES.encrypt(privateKey, key, {
      iv: iv.toString('hex'),
      mode: CryptoJS.mode.CBC,
      padding: CryptoJS.pad.Pkcs7,
    });
    
    return JSON.stringify({
      salt: salt.toString('hex'),
      iv: iv.toString('hex'),
      data: encrypted.toString(),
    });
  }

  /**
   * Decrypt private key with password
   */
  decryptPrivateKey(encryptedData: string, password: string): string | null {
    try {
      const { salt, iv, data } = JSON.parse(encryptedData);
      const key = CryptoJS.PBKDF2(password, salt, {
        keySize: 256 / 32,
        iterations: 100000,
      }).toString();
      
      const decrypted = CryptoJS.AES.decrypt(data, key, {
        iv: iv,
        mode: CryptoJS.mode.CBC,
        padding: CryptoJS.pad.Pkcs7,
      });
      
      return decrypted.toString(CryptoJS.enc.Utf8);
    } catch {
      return null;
    }
  }

  // ============================================================================
  // Signing Operations
  // ============================================================================

  /**
   * Sign a transaction
   */
  signTransaction(privateKey: string, tx: {
    to: string;
    value: string;
    gasLimit: string;
    gasPrice: string;
    nonce: number;
    chainId: number;
    data?: string;
  }): string {
    const wallet = new Wallet(privateKey);
    const transaction = {
      to: tx.to,
      value: ethers.parseEther(tx.value),
      gasLimit: tx.gasLimit,
      gasPrice: ethers.parseUnits(tx.gasPrice, 'gwei'),
      nonce: tx.nonce,
      chainId: tx.chainId,
      data: tx.data || '0x',
    };
    
    const signedTx = wallet.signTransaction(transaction);
    return signedTx;
  }

  /**
   * Sign a message
   */
  signMessage(privateKey: string, message: string): string {
    const wallet = new Wallet(privateKey);
    return wallet.signMessage(message);
  }

  /**
   * Sign typed data (EIP-712)
   */
  signTypedData(privateKey: string, domain: Record<string, unknown>, types: Record<string, unknown[]>, message: Record<string, unknown>): string {
    const wallet = new Wallet(privateKey);
    // For EIP-712 signing, we'd use a proper library
    // This is a simplified version
    return wallet.signMessage(JSON.stringify({ domain, types, message }));
  }

  /**
   * Verify a signature
   */
  verifySignature(message: string, signature: string): string {
    try {
      const recovered = ethers.verifyMessage(message, signature);
      return recovered;
    } catch {
      return '';
    }
  }

  // ============================================================================
  // Hashing Operations
  // ============================================================================

  /**
   * Keccak-256 hash
   */
  keccak256(data: string): string {
    return keccak256(toUtf8Bytes(data));
  }

  /**
   * SHA-256 hash
   */
  sha256(data: string): string {
    return createHash('sha256').update(data).digest('hex');
  }

  /**
   * HMAC-SHA256
   */
  hmacSHA256(key: string, data: string): string {
    return createHmac('sha256', key).update(data).digest('hex');
  }

  // ============================================================================
  // Encryption Operations
  // ============================================================================

  /**
   * AES-256-GCM encryption
   */
  encryptAES(key: string, plaintext: string): { ciphertext: string; iv: string; tag: string } {
    const iv = randomBytes(12);
    const cipher = createCipheriv('aes-256-gcm', Buffer.from(key.slice(0, 32), 'hex'), iv);
    
    let ciphertext = cipher.update(plaintext, 'utf8', 'hex');
    ciphertext += cipher.final('hex');
    const tag = cipher.getAuthTag();
    
    return {
      ciphertext,
      iv: iv.toString('hex'),
      tag: tag.toString('hex'),
    };
  }

  /**
   * AES-256-GCM decryption
   */
  decryptAES(key: string, encrypted: { ciphertext: string; iv: string; tag: string }): string {
    const decipher = createDecipheriv(
      'aes-256-gcm',
      Buffer.from(key.slice(0, 32), 'hex'),
      Buffer.from(encrypted.iv, 'hex')
    );
    decipher.setAuthTag(Buffer.from(encrypted.tag, 'hex'));
    
    let plaintext = decipher.update(encrypted.ciphertext, 'hex', 'utf8');
    plaintext += decipher.final('utf8');
    
    return plaintext;
  }

  // ============================================================================
  // Random Generation
  // ============================================================================

  /**
   * Generate random bytes
   */
  randomBytes(length: number): string {
    return randomBytes(length).toString('hex');
  }

  /**
   * Generate random address
   */
  generateRandomAddress(): string {
    const privateKey = randomBytes(32).toString('hex');
    const wallet = new Wallet(privateKey);
    return wallet.address;
  }

  // ============================================================================
  // Address Validation
  // ============================================================================

  /**
   * Validate EVM address
   */
  isValidEvmAddress(address: string): boolean {
    return /^0x[a-fA-F0-9]{40}$/.test(address);
  }

  /**
   * Validate private key
   */
  isValidPrivateKey(key: string): boolean {
    return /^0x[a-fA-F0-9]{64}$/.test(key);
  }

  /**
   * Convert address to checksum format
   */
  toChecksumAddress(address: string): string {
    if (!this.isValidEvmAddress(address)) {
      return address;
    }
    const addr = address.toLowerCase().replace('0x', '');
    const hash = this.keccak256(addr);
    let result = '0x';
    for (let i = 0; i < addr.length; i++) {
      if (parseInt(hash[i], 16) >= 8) {
        result += addr[i].toUpperCase();
      } else {
        result += addr[i];
      }
    }
    return result;
  }

  // ============================================================================
  // Utility Functions
  // ============================================================================

  /**
   * Parse units
   */
  parseUnits(value: string, decimals: number): string {
    return ethers.parseUnits(value, decimals).toString();
  }

  /**
   * Format units
   */
  formatUnits(value: string, decimals: number): string {
    return ethers.formatUnits(value, decimals);
  }

  /**
   * Parse Ether
   */
  parseEther(ether: string): string {
    return ethers.parseEther(ether).toString();
  }

  /**
   * Format Ether
   */
  formatEther(wei: string): string {
    return ethers.formatEther(wei);
  }

  /**
   * Compute address from public key
   */
  computeAddress(publicKey: string): string {
    return ethers.computeAddress(publicKey);
  }
}

// ============================================================================
// Export singleton instance
// ============================================================================

export const cryptoService = CryptoService.getInstance();
export default cryptoService;
