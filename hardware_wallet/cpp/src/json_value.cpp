#include "json_value.h"

#include <cmath>
#include <cstdlib>
#include <limits>

namespace tigerwallet {
namespace json {

/// A recursive-descent reader over a JSON document.
class Parser {
public:
    explicit Parser(const std::string& text) : text_(text) {}

    Value parse_document() {
        skip_whitespace();
        Value value = parse_value();
        skip_whitespace();

        if (position_ != text_.size()) {
            throw ParseError("the document has trailing content at offset " +
                             std::to_string(position_));
        }

        return value;
    }

private:
    void skip_whitespace() {
        while (position_ < text_.size()) {
            const char c = text_[position_];
            if (c == ' ' || c == '\t' || c == '\n' || c == '\r') {
                position_++;
            } else {
                break;
            }
        }
    }

    char peek() const {
        if (position_ >= text_.size()) {
            throw ParseError("the document ended early");
        }
        return text_[position_];
    }

    void expect(char c) {
        if (peek() != c) {
            throw ParseError(std::string("expected '") + c + "' at offset " +
                             std::to_string(position_));
        }
        position_++;
    }

    Value parse_value() {
        skip_whitespace();

        if (position_ >= text_.size()) {
            throw ParseError("the document ended before a value");
        }

        const char c = text_[position_];

        switch (c) {
            case '{':
                return parse_object();
            case '[':
                return parse_array();
            case '"':
                return parse_string_value();
            case 't':
                expect_literal("true");
                return make_boolean(true);
            case 'f':
                expect_literal("false");
                return make_boolean(false);
            case 'n':
                expect_literal("null");
                return make_null();
            default:
                if (c == '-' || (c >= '0' && c <= '9')) {
                    return parse_number();
                }
                throw ParseError(std::string("unexpected character '") + c +
                                 "' at offset " + std::to_string(position_));
        }
    }

    void expect_literal(const char* literal) {
        const size_t length = std::char_traits<char>::length(literal);
        if (text_.compare(position_, length, literal) != 0) {
            throw ParseError(std::string("expected '") + literal + "' at offset " +
                             std::to_string(position_));
        }
        position_ += length;
    }

    static Value make_boolean(bool value) {
        Value result;
        result.type_ = Value::Type::Boolean;
        result.boolean_ = value;
        return result;
    }

    static Value make_null() {
        Value result;
        result.type_ = Value::Type::Null;
        return result;
    }

    Value parse_object() {
        expect('{');
        Value result;
        result.type_ = Value::Type::Object;

        skip_whitespace();
        if (peek() == '}') {
            position_++;
            return result;
        }

        while (true) {
            skip_whitespace();
            const std::string key = parse_string();
            skip_whitespace();
            expect(':');

            Value value = parse_value();

            // A repeated key is a malformed document, not something to
            // silently overwrite.
            if (result.object_.count(key) != 0) {
                throw ParseError("the object repeats the key \"" + key + "\"");
            }
            result.object_.emplace(key, std::move(value));

            skip_whitespace();
            const char next = peek();
            if (next == ',') {
                position_++;
                continue;
            }
            if (next == '}') {
                position_++;
                return result;
            }
            throw ParseError("expected ',' or '}' at offset " + std::to_string(position_));
        }
    }

    Value parse_array() {
        expect('[');
        Value result;
        result.type_ = Value::Type::Array;

        skip_whitespace();
        if (peek() == ']') {
            position_++;
            return result;
        }

        while (true) {
            result.array_.push_back(parse_value());

            skip_whitespace();
            const char next = peek();
            if (next == ',') {
                position_++;
                continue;
            }
            if (next == ']') {
                position_++;
                return result;
            }
            throw ParseError("expected ',' or ']' at offset " + std::to_string(position_));
        }
    }

    Value parse_string_value() {
        Value result;
        result.type_ = Value::Type::String;
        result.string_ = parse_string();
        return result;
    }

    /// Appends one code point as UTF-8.
    static void append_utf8(std::string& out, uint32_t code_point) {
        if (code_point <= 0x7F) {
            out.push_back(static_cast<char>(code_point));
        } else if (code_point <= 0x7FF) {
            out.push_back(static_cast<char>(0xC0 | (code_point >> 6)));
            out.push_back(static_cast<char>(0x80 | (code_point & 0x3F)));
        } else if (code_point <= 0xFFFF) {
            out.push_back(static_cast<char>(0xE0 | (code_point >> 12)));
            out.push_back(static_cast<char>(0x80 | ((code_point >> 6) & 0x3F)));
            out.push_back(static_cast<char>(0x80 | (code_point & 0x3F)));
        } else {
            out.push_back(static_cast<char>(0xF0 | (code_point >> 18)));
            out.push_back(static_cast<char>(0x80 | ((code_point >> 12) & 0x3F)));
            out.push_back(static_cast<char>(0x80 | ((code_point >> 6) & 0x3F)));
            out.push_back(static_cast<char>(0x80 | (code_point & 0x3F)));
        }
    }

    uint32_t parse_hex4() {
        uint32_t value = 0;
        for (int i = 0; i < 4; i++) {
            if (position_ >= text_.size()) {
                throw ParseError("the \\u escape is truncated");
            }
            const char c = text_[position_++];
            value <<= 4;
            if (c >= '0' && c <= '9') {
                value |= static_cast<uint32_t>(c - '0');
            } else if (c >= 'a' && c <= 'f') {
                value |= static_cast<uint32_t>(c - 'a' + 10);
            } else if (c >= 'A' && c <= 'F') {
                value |= static_cast<uint32_t>(c - 'A' + 10);
            } else {
                throw ParseError("the \\u escape has a non-hex digit");
            }
        }
        return value;
    }

    std::string parse_string() {
        expect('"');

        std::string out;

        while (true) {
            if (position_ >= text_.size()) {
                throw ParseError("the string is not closed");
            }

            const unsigned char c = static_cast<unsigned char>(text_[position_++]);

            if (c == '"') {
                return out;
            }

            if (c < 0x20) {
                throw ParseError("the string holds a raw control character");
            }

            if (c != '\\') {
                out.push_back(static_cast<char>(c));
                continue;
            }

            if (position_ >= text_.size()) {
                throw ParseError("the escape is truncated");
            }

            const char escape = text_[position_++];
            switch (escape) {
                case '"': out.push_back('"'); break;
                case '\\': out.push_back('\\'); break;
                case '/': out.push_back('/'); break;
                case 'b': out.push_back('\b'); break;
                case 'f': out.push_back('\f'); break;
                case 'n': out.push_back('\n'); break;
                case 'r': out.push_back('\r'); break;
                case 't': out.push_back('\t'); break;
                case 'u': {
                    uint32_t code_point = parse_hex4();

                    // A high surrogate must be followed by a low surrogate.
                    if (code_point >= 0xD800 && code_point <= 0xDBFF) {
                        if (position_ + 1 >= text_.size() ||
                            text_[position_] != '\\' || text_[position_ + 1] != 'u') {
                            throw ParseError("a high surrogate is not followed by a low surrogate");
                        }
                        position_ += 2;
                        const uint32_t low = parse_hex4();
                        if (low < 0xDC00 || low > 0xDFFF) {
                            throw ParseError("the low surrogate is out of range");
                        }
                        code_point = 0x10000 + ((code_point - 0xD800) << 10) + (low - 0xDC00);
                    } else if (code_point >= 0xDC00 && code_point <= 0xDFFF) {
                        throw ParseError("a low surrogate appears on its own");
                    }

                    append_utf8(out, code_point);
                    break;
                }
                default:
                    throw ParseError(std::string("unknown escape '\\") + escape + "'");
            }
        }
    }

    Value parse_number() {
        const size_t start = position_;

        if (peek() == '-') {
            position_++;
        }

        // An integer part is either a single zero or a non-zero digit run.
        if (position_ >= text_.size()) {
            throw ParseError("the number is truncated");
        }
        if (text_[position_] == '0') {
            position_++;
        } else if (text_[position_] >= '1' && text_[position_] <= '9') {
            while (position_ < text_.size() &&
                   text_[position_] >= '0' && text_[position_] <= '9') {
                position_++;
            }
        } else {
            throw ParseError("the number has no integer part");
        }

        if (position_ < text_.size() && text_[position_] == '.') {
            position_++;
            if (position_ >= text_.size() ||
                text_[position_] < '0' || text_[position_] > '9') {
                throw ParseError("the number has no fraction digits");
            }
            while (position_ < text_.size() &&
                   text_[position_] >= '0' && text_[position_] <= '9') {
                position_++;
            }
        }

        if (position_ < text_.size() &&
            (text_[position_] == 'e' || text_[position_] == 'E')) {
            position_++;
            if (position_ < text_.size() &&
                (text_[position_] == '+' || text_[position_] == '-')) {
                position_++;
            }
            if (position_ >= text_.size() ||
                text_[position_] < '0' || text_[position_] > '9') {
                throw ParseError("the exponent has no digits");
            }
            while (position_ < text_.size() &&
                   text_[position_] >= '0' && text_[position_] <= '9') {
                position_++;
            }
        }

        const std::string literal = text_.substr(start, position_ - start);

        errno = 0;
        char* end = nullptr;
        const double parsed = std::strtod(literal.c_str(), &end);

        if (end != literal.c_str() + literal.size() || errno == ERANGE ||
            !std::isfinite(parsed)) {
            throw ParseError("the number \"" + literal + "\" is out of range");
        }

        Value result;
        result.type_ = Value::Type::Number;
        result.number_ = parsed;
        return result;
    }

    const std::string& text_;
    size_t position_ = 0;
};

bool Value::as_boolean() const {
    if (type_ != Type::Boolean) {
        throw ParseError("the value is not a boolean");
    }
    return boolean_;
}

double Value::as_number() const {
    if (type_ != Type::Number) {
        throw ParseError("the value is not a number");
    }
    return number_;
}

const std::string& Value::as_string() const {
    if (type_ != Type::String) {
        throw ParseError("the value is not a string");
    }
    return string_;
}

const std::vector<Value>& Value::as_array() const {
    if (type_ != Type::Array) {
        throw ParseError("the value is not an array");
    }
    return array_;
}

const std::map<std::string, Value>& Value::as_object() const {
    if (type_ != Type::Object) {
        throw ParseError("the value is not an object");
    }
    return object_;
}

const Value* Value::find(const std::string& key) const {
    if (type_ != Type::Object) {
        return nullptr;
    }

    auto it = object_.find(key);
    return (it == object_.end()) ? nullptr : &it->second;
}

const Value& Value::at(const std::string& key) const {
    const Value* found = find(key);
    if (found == nullptr) {
        throw ParseError("the object has no member \"" + key + "\"");
    }
    return *found;
}

std::string Value::string_at(const std::string& key) const {
    return at(key).as_string();
}

const Value& Value::at(size_t index) const {
    if (type_ != Type::Array) {
        throw ParseError("the value is not an array");
    }
    if (index >= array_.size()) {
        throw ParseError("the array index is out of range");
    }
    return array_[index];
}

size_t Value::size() const {
    if (type_ == Type::Array) {
        return array_.size();
    }
    if (type_ == Type::Object) {
        return object_.size();
    }
    throw ParseError("the value has no size");
}

Value Value::parse(const std::string& text) {
    Parser parser(text);
    return parser.parse_document();
}

}  // namespace json
}  // namespace tigerwallet
