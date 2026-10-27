/**
 * A small, strict JSON reader.
 *
 * The hardware wallet module talks to EVM nodes over JSON-RPC. Rather than
 * pull in a dependency for that, this header provides just enough of RFC 8259
 * to read a response. It is deliberately strict: malformed input is an error
 * rather than something to guess at.
 */

#ifndef TIGERWALLET_JSON_VALUE_H
#define TIGERWALLET_JSON_VALUE_H

#include <cstdint>
#include <map>
#include <stdexcept>
#include <string>
#include <vector>

namespace tigerwallet {
namespace json {

class ParseError : public std::runtime_error {
public:
    explicit ParseError(const std::string& what) : std::runtime_error(what) {}
};

class Value {
public:
    enum class Type { Null, Boolean, Number, String, Array, Object };

    Value() : type_(Type::Null) {}

    Type type() const { return type_; }
    bool is_null() const { return type_ == Type::Null; }
    bool is_boolean() const { return type_ == Type::Boolean; }
    bool is_number() const { return type_ == Type::Number; }
    bool is_string() const { return type_ == Type::String; }
    bool is_array() const { return type_ == Type::Array; }
    bool is_object() const { return type_ == Type::Object; }

    /// The value as a boolean. Throws when the type does not match.
    bool as_boolean() const;

    /// The value as a number. Throws when the type does not match.
    double as_number() const;

    /// The value as a string. Throws when the type does not match.
    const std::string& as_string() const;

    /// The array elements. Throws when the type does not match.
    const std::vector<Value>& as_array() const;

    /// The object members. Throws when the type does not match.
    const std::map<std::string, Value>& as_object() const;

    /// Looks a member up in an object. Returns nullptr when the key is absent
    /// or the value is not an object.
    const Value* find(const std::string& key) const;

    /// Looks a member up and throws when it is missing.
    const Value& at(const std::string& key) const;

    /// Reads a string member, throwing when it is missing or not a string.
    std::string string_at(const std::string& key) const;

    /// Reads the element at an index, throwing when it is out of range.
    const Value& at(size_t index) const;

    size_t size() const;

    /// Parses a complete document. Trailing content other than whitespace is
    /// an error, so a truncated or concatenated response is not accepted.
    static Value parse(const std::string& text);

private:
    friend class Parser;

    Type type_;
    bool boolean_ = false;
    double number_ = 0.0;
    std::string string_;
    std::vector<Value> array_;
    std::map<std::string, Value> object_;
};

}  // namespace json
}  // namespace tigerwallet

#endif  // TIGERWALLET_JSON_VALUE_H
