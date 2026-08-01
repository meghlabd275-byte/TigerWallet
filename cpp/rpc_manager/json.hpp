/**
 * Minimal JSON Header-Only Library for RPC Manager
 * Compatible with nlohmann/json API
 */

#ifndef TIGERWALLET_JSON_HPP
#define TIGERWALLET_JSON_HPP

#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <variant>
#include <optional>
#include <initializer_list>
#include <memory>
#include <type_traits>

namespace nlohmann {

// Forward declarations
class json;

// ============================================================================
// JSON Value Types
// ============================================================================

using json_value = std::variant<
    std::nullptr_t,
    bool,
    int64_t,
    double,
    std::string,
    std::vector<json>,
    std::map<std::string, json>
>;

// ============================================================================
// JSON Class
// ============================================================================

class json {
private:
    json_value value_;
    static constexpr size_t NULL_INDEX = 0;
    static constexpr size_t BOOL_INDEX = 1;
    static constexpr size_t INT_INDEX = 2;
    static constexpr size_t DOUBLE_INDEX = 3;
    static constexpr size_t STRING_INDEX = 4;
    static constexpr size_t ARRAY_INDEX = 5;
    static constexpr size_t OBJECT_INDEX = 6;

public:
    // Constructors
    json() : value_(nullptr) {}
    
    json(std::nullptr_t) : value_(nullptr) {}
    
    json(bool b) : value_(b) {}
    
    json(int i) : value_(static_cast<int64_t>(i)) {}
    
    json(int64_t i) : value_(i) {}
    
    json(double d) : value_(d) {}
    
    json(const char* s) : value_(std::string(s)) {}
    
    json(const std::string& s) : value_(s) {}
    
    json(std::initializer_list<json> init) {
        if (init.size() == 0) {
            value_ = nullptr;
        } else {
            // Check if it's an object (pairs) or array
            bool is_object = true;
            size_t i = 0;
            for (const auto& item : init) {
                if (i % 2 == 0) {
                    if (!item.is_string()) {
                        is_object = false;
                        break;
                    }
                }
                i++;
            }
            
            if (is_object && init.size() % 2 == 0) {
                std::map<std::string, json> obj;
                auto it = init.begin();
                while (it != init.end()) {
                    std::string key = it->get<std::string>();
                    ++it;
                    if (it != init.end()) {
                        obj[key] = *it;
                        ++it;
                    }
                }
                value_ = obj;
            } else {
                std::vector<json> arr;
                for (const auto& item : init) {
                    arr.push_back(item);
                }
                value_ = arr;
            }
        }
    }
    
    // Array constructor
    static json array() {
        return json(std::vector<json>{});
    }
    
    static json array(std::initializer_list<json> init) {
        return json(init);
    }
    
    // Object constructor
    static json object() {
        return json(std::map<std::string, json>{});
    }
    
    // Type checking
    bool is_null() const { return std::holds_alternative<std::nullptr_t>(value_); }
    bool is_bool() const { return std::holds_alternative<bool>(value_); }
    bool is_number() const { 
        return std::holds_alternative<int64_t>(value_) || 
               std::holds_alternative<double>(value_); 
    }
    bool is_integer() const { return std::holds_alternative<int64_t>(value_); }
    bool is_float() const { return std::holds_alternative<double>(value_); }
    bool is_string() const { return std::holds_alternative<std::string>(value_); }
    bool is_array() const { return std::holds_alternative<std::vector<json>>(value_); }
    bool is_object() const { return std::holds_alternative<std::map<std::string, json>>(value_); }
    
    // Type checking helpers
    bool is_primitive() const {
        return is_null() || is_bool() || is_number() || is_string();
    }
    
    // Get values
    template<typename T>
    T get() const {
        if constexpr (std::is_same_v<T, bool>) {
            return std::get<bool>(value_);
        } else if constexpr (std::is_same_v<T, int64_t>) {
            return std::get<int64_t>(value_);
        } else if constexpr (std::is_same_v<T, int>) {
            return static_cast<int>(std::get<int64_t>(value_));
        } else if constexpr (std::is_same_v<T, double>) {
            if (std::holds_alternative<double>(value_)) {
                return std::get<double>(value_);
            }
            return static_cast<double>(std::get<int64_t>(value_));
        } else if constexpr (std::is_same_v<T, std::string>) {
            return std::get<std::string>(value_);
        } else if constexpr (std::is_same_v<T, std::vector<json>>) {
            return std::get<std::vector<json>>(value_);
        } else if constexpr (std::is_same_v<T, std::map<std::string, json>>) {
            return std::get<std::map<std::string, json>>(value_);
        }
        throw std::runtime_error("Invalid type");
    }
    
    template<typename T>
    T get_or(const T& default_value) const {
        try {
            return get<T>();
        } catch (...) {
            return default_value;
        }
    }
    
    // Accessors
    json& operator[](const std::string& key) {
        auto& obj = std::get<std::map<std::string, json>>(value_);
        return obj[key];
    }
    
    const json& operator[](const std::string& key) const {
        static json null_json;
        const auto& obj = std::get<std::map<std::string, json>>(value_);
        auto it = obj.find(key);
        return (it != obj.end()) ? it->second : null_json;
    }
    
    json& operator[](size_t index) {
        auto& arr = std::get<std::vector<json>>(value_);
        return arr[index];
    }
    
    const json& operator[](size_t index) const {
        static json null_json;
        const auto& arr = std::get<std::vector<json>>(value_);
        return index < arr.size() ? arr[index] : null_json;
    }
    
    // Check if key exists
    bool contains(const std::string& key) const {
        if (!is_object()) return false;
        const auto& obj = std::get<std::map<std::string, json>>(value_);
        return obj.find(key) != obj.end();
    }
    
    // Size
    size_t size() const {
        if (is_array()) {
            return std::get<std::vector<json>>(value_).size();
        } else if (is_object()) {
            return std::get<std::map<std::string, json>>(value_).size();
        }
        return 0;
    }
    
    bool empty() const {
        return size() == 0;
    }
    
    // Iteration
    class iterator {
    public:
        using iterator_category = std::forward_iterator_tag;
        using value_type = json;
        using difference_type = std::ptrdiff_t;
        using pointer = json*;
        using reference = json&;
        
    private:
        std::variant<
            std::vector<json>::iterator,
            std::map<std::string, json>::iterator
        > iter_;
        bool is_array_;
        
    public:
        iterator() : is_array_(false) {}
        
        iterator(const std::vector<json>::iterator& it) 
            : iter_(it), is_array_(true) {}
        
        iterator(const std::map<std::string, json>::iterator& it) 
            : iter_(it), is_array_(false) {}
        
        reference operator*() {
            if (is_array_) {
                return *std::get<std::vector<json>::iterator>(iter_);
            }
            return std::get<std::map<std::string, json>::iterator>(iter_)->second;
        }
        
        pointer operator->() {
            if (is_array_) {
                return &*std::get<std::vector<json>::iterator>(iter_);
            }
            return &std::get<std::map<std::string, json>::iterator>(iter_)->second;
        }
        
        iterator& operator++() {
            if (is_array_) {
                ++std::get<std::vector<json>::iterator>(iter_);
            } else {
                ++std::get<std::map<std::string, json>::iterator>(iter_);
            }
            return *this;
        }
        
        iterator operator++(int) {
            iterator tmp = *this;
            ++(*this);
            return tmp;
        }
        
        bool operator==(const iterator& other) const {
            if (is_array_ != other.is_array_) return false;
            if (is_array_) {
                return std::get<std::vector<json>::iterator>(iter_) == 
                       std::get<std::vector<json>::iterator>(other.iter_);
            }
            return std::get<std::map<std::string, json>::iterator>(iter_) ==
                   std::get<std::map<std::string, json>::iterator>(other.iter_);
        }
        
        bool operator!=(const iterator& other) const {
            return !(*this == other);
        }
    };
    
    iterator begin() {
        if (is_array()) {
            return iterator(std::get<std::vector<json>>(value_).begin());
        } else if (is_object()) {
            return iterator(std::get<std::map<std::string, json>>(value_).begin());
        }
        return iterator();
    }
    
    iterator end() {
        if (is_array()) {
            return iterator(std::get<std::vector<json>>(value_).end());
        } else if (is_object()) {
            return iterator(std::get<std::map<std::string, json>>(value_).end());
        }
        return iterator();
    }
    
    // Serialization
    std::string dump() const {
        return dump_internal(0);
    }
    
    std::string dump(int indent) const {
        return dump_internal(indent);
    }
    
private:
    std::string dump_internal(int indent) const {
        std::ostringstream oss;
        
        if (is_null()) {
            oss << "null";
        } else if (is_bool()) {
            oss << (get<bool>() ? "true" : "false");
        } else if (is_integer()) {
            oss << get<int64_t>();
        } else if (is_float()) {
            oss << std::fixed << std::setprecision(8) << get<double>();
        } else if (is_string()) {
            oss << "\"" << escape_string(get<std::string>()) << "\"";
        } else if (is_array()) {
            oss << "[";
            const auto& arr = get<std::vector<json>>();
            for (size_t i = 0; i < arr.size(); i++) {
                if (i > 0) oss << ",";
                oss << arr[i].dump_internal(indent);
            }
            oss << "]";
        } else if (is_object()) {
            oss << "{";
            const auto& obj = get<std::map<std::string, json>>();
            bool first = true;
            for (const auto& [key, value] : obj) {
                if (!first) oss << ",";
                first = false;
                oss << "\"" << escape_string(key) << "\":" 
                    << value.dump_internal(indent);
            }
            oss << "}";
        }
        
        return oss.str();
    }
    
    static std::string escape_string(const std::string& s) {
        std::ostringstream oss;
        for (char c : s) {
            switch (c) {
                case '"': oss << "\\\""; break;
                case '\\': oss << "\\\\"; break;
                case '\n': oss << "\\n"; break;
                case '\r': oss << "\\r"; break;
                case '\t': oss << "\\t"; break;
                default: oss << c; break;
            }
        }
        return oss.str();
    }
    
public:
    // Parse JSON string
    static json parse(const std::string& str) {
        size_t pos = 0;
        return parse_value(str, pos);
    }
    
private:
    static json parse_value(const std::string& s, size_t& pos) {
        skip_whitespace(s, pos);
        
        if (pos >= s.length()) {
            return json::object();
        }
        
        char c = s[pos];
        
        if (c == '{') {
            return parse_object(s, pos);
        } else if (c == '[') {
            return parse_array(s, pos);
        } else if (c == '"') {
            return parse_string(s, pos);
        } else if (c == 't' && s.substr(pos, 4) == "true") {
            pos += 4;
            return json(true);
        } else if (c == 'f' && s.substr(pos, 5) == "false") {
            pos += 5;
            return json(false);
        } else if (c == 'n' && s.substr(pos, 4) == "null") {
            pos += 4;
            return json(nullptr);
        } else if (c == '-' || (c >= '0' && c <= '9')) {
            return parse_number(s, pos);
        }
        
        return json::object();
    }
    
    static json parse_object(const std::string& s, size_t& pos) {
        pos++; // skip '{'
        skip_whitespace(s, pos);
        
        std::map<std::string, json> obj;
        
        if (pos < s.length() && s[pos] == '}') {
            pos++;
            return obj;
        }
        
        while (pos < s.length()) {
            skip_whitespace(s, pos);
            
            if (s[pos] != '"') break;
            
            json key = parse_string(s, pos);
            skip_whitespace(s, pos);
            
            if (pos < s.length() && s[pos] == ':') {
                pos++;
            }
            
            json value = parse_value(s, pos);
            obj[key.get<std::string>()] = value;
            
            skip_whitespace(s, pos);
            
            if (pos < s.length() && s[pos] == ',') {
                pos++;
            } else if (pos < s.length() && s[pos] == '}') {
                pos++;
                break;
            }
        }
        
        return obj;
    }
    
    static json parse_array(const std::string& s, size_t& pos) {
        pos++; // skip '['
        skip_whitespace(s, pos);
        
        std::vector<json> arr;
        
        if (pos < s.length() && s[pos] == ']') {
            pos++;
            return arr;
        }
        
        while (pos < s.length()) {
            json value = parse_value(s, pos);
            arr.push_back(value);
            
            skip_whitespace(s, pos);
            
            if (pos < s.length() && s[pos] == ',') {
                pos++;
            } else if (pos < s.length() && s[pos] == ']') {
                pos++;
                break;
            }
        }
        
        return arr;
    }
    
    static json parse_string(const std::string& s, size_t& pos) {
        pos++; // skip '"'
        std::string str;
        
        while (pos < s.length() && s[pos] != '"') {
            if (s[pos] == '\\' && pos + 1 < s.length()) {
                pos++;
                switch (s[pos]) {
                    case 'n': str += '\n'; break;
                    case 'r': str += '\r'; break;
                    case 't': str += '\t'; break;
                    case '"': str += '"'; break;
                    case '\\': str += '\\'; break;
                    default: str += s[pos]; break;
                }
            } else {
                str += s[pos];
            }
            pos++;
        }
        
        if (pos < s.length()) pos++; // skip closing '"'
        
        return str;
    }
    
    static json parse_number(const std::string& s, size_t& pos) {
        size_t start = pos;
        
        if (s[pos] == '-') pos++;
        
        // Integer part
        while (pos < s.length() && s[pos] >= '0' && s[pos] <= '9') {
            pos++;
        }
        
        // Decimal part
        if (pos < s.length() && s[pos] == '.') {
            pos++;
            while (pos < s.length() && s[pos] >= '0' && s[pos] <= '9') {
                pos++;
            }
        }
        
        // Exponent
        if (pos < s.length() && (s[pos] == 'e' || s[pos] == 'E')) {
            pos++;
            if (pos < s.length() && (s[pos] == '+' || s[pos] == '-')) {
                pos++;
            }
            while (pos < s.length() && s[pos] >= '0' && s[pos] <= '9') {
                pos++;
            }
        }
        
        std::string num_str = s.substr(start, pos - start);
        
        // Check if it's integer or float
        if (num_str.find('.') == std::string::npos && 
            num_str.find('e') == std::string::npos &&
            num_str.find('E') == std::string::npos) {
            return json(std::stoll(num_str));
        }
        
        return json(std::stod(num_str));
    }
    
    static void skip_whitespace(const std::string& s, size_t& pos) {
        while (pos < s.length() && 
               (s[pos] == ' ' || s[pos] == '\n' || 
                s[pos] == '\r' || s[pos] == '\t')) {
            pos++;
        }
    }
};

// Stream operators
inline std::ostream& operator<<(std::ostream& os, const json& j) {
    os << j.dump();
    return os;
}

} // namespace nlohmann

#endif // TIGERWALLET_JSON_HPP
