/**
 * TigerWallet Logo Upload Service - Implementation
 * High-performance C++ service for token logo upload, processing, and CDN distribution
 */

#include "logo_upload_service.hpp"
#include <algorithm>
#include <random>
#include <iomanip>
#include <sstream>

// ============================================================================
// Image Processor Implementation
// ============================================================================

std::string ImageProcessor::computeHash(const std::vector<unsigned char>& data) {
    unsigned char hash[SHA256_DIGEST_LENGTH];
    SHA256(data.data(), data.size(), hash);
    
    std::stringstream ss;
    for (int i = 0; i < SHA256_DIGEST_LENGTH; i++) {
        ss << std::hex << std::setw(2) << std::setfill('0') << (int)hash[i];
    }
    return ss.str();
}

std::string ImageProcessor::getImageFormat(const std::vector<unsigned char>& data) {
    if (data.size() < 4) return "unknown";
    
    // PNG: 89 50 4E 47
    if (data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47) {
        return "png";
    }
    
    // JPEG: FF D8 FF
    if (data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF) {
        return "jpeg";
    }
    
    // GIF: 47 49 46 38
    if (data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38) {
        return "gif";
    }
    
    // WebP: 52 49 46 46 ... 57 45 42 50
    if (data.size() >= 12 && data[0] == 0x52 && data[1] == 0x49 && 
        data[2] == 0x46 && data[3] == 0x46) {
        if (data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50) {
            return "webp";
        }
    }
    
    return "unknown";
}

bool ImageProcessor::isPNG(const std::vector<unsigned char>& data) {
    return getImageFormat(data) == "png";
}

bool ImageProcessor::isJPEG(const std::vector<unsigned char>& data) {
    return getImageFormat(data) == "jpeg";
}

bool ImageProcessor::isGIF(const std::vector<unsigned char>& data) {
    return getImageFormat(data) == "gif";
}

bool ImageProcessor::isWebP(const std::vector<unsigned char>& data) {
    return getImageFormat(data) == "webp";
}

bool ImageProcessor::validateImage(
    const std::vector<unsigned char>& data,
    std::string& mime_type,
    int& width,
    int& height
) {
    if (data.size() < 12) {
        return false;
    }
    
    std::string format = getImageFormat(data);
    
    if (format == "png") {
        mime_type = "image/png";
        // Read PNG dimensions from IHDR chunk
        if (data.size() >= 24) {
            width = (data[16] << 24) | (data[17] << 16) | (data[18] << 8) | data[19];
            height = (data[20] << 24) | (data[21] << 16) | (data[22] << 8) | data[23];
        }
        return true;
    }
    
    if (format == "jpeg") {
        mime_type = "image/jpeg";

        // Walk the JPEG marker segments to the SOFn frame header, which
        // carries the real dimensions. Scanning is required because the
        // frame header sits after a variable number of segments.
        size_t pos = 2;  // skip SOI
        while (pos + 3 < data.size()) {
            if (data[pos] != 0xFF) {
                return false;  // lost marker alignment
            }

            unsigned char marker = data[pos + 1];
            if (marker == 0xFF) {
                ++pos;  // fill byte
                continue;
            }

            // Standalone markers carry no length.
            if (marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD9)) {
                pos += 2;
                continue;
            }

            unsigned int segment_len =
                (static_cast<unsigned int>(data[pos + 2]) << 8) | data[pos + 3];
            if (segment_len < 2 || pos + 2 + segment_len > data.size()) {
                return false;  // truncated segment
            }

            // SOF0..SOF15, excluding DHT (C4), JPG (C8) and DAC (CC).
            bool is_sof = (marker >= 0xC0 && marker <= 0xCF) &&
                          marker != 0xC4 && marker != 0xC8 && marker != 0xCC;
            if (is_sof) {
                if (segment_len < 7) {
                    return false;
                }
                height = (static_cast<int>(data[pos + 5]) << 8) | data[pos + 6];
                width = (static_cast<int>(data[pos + 7]) << 8) | data[pos + 8];
                return width > 0 && height > 0;
            }

            // Start of scan: no frame header was found.
            if (marker == 0xDA) {
                return false;
            }

            pos += 2 + segment_len;
        }

        return false;
    }
    
    if (format == "gif") {
        mime_type = "image/gif";
        if (data.size() < 10) {
            return false;
        }
        // Logical screen descriptor: little-endian dimensions.
        width = data[6] | (data[7] << 8);
        height = data[8] | (data[9] << 8);
        return width > 0 && height > 0;
    }
    
    if (format == "webp") {
        mime_type = "image/webp";

        // Only VP8X, VP8 (lossy) and VP8L (lossless) carry dimensions in
        // headers this codec understands; anything else is refused rather
        // than reported as valid with unknown dimensions.
        std::string fourcc(reinterpret_cast<const char*>(&data[12]), 4);

        if (fourcc == "VP8X") {
            if (data.size() < 30) {
                return false;
            }
            width = 1 + (data[24] | (data[25] << 8) | (data[26] << 16));
            height = 1 + (data[27] | (data[28] << 8) | (data[29] << 16));
        } else if (fourcc == "VP8 ") {
            if (data.size() < 30) {
                return false;
            }
            // Keyframe: 3-byte frame tag, then the 0x9D012A start code.
            width = ((data[27] << 8) | data[26]) & 0x3FFF;
            height = ((data[29] << 8) | data[28]) & 0x3FFF;
        } else if (fourcc == "VP8L") {
            if (data.size() < 25) {
                return false;
            }
            unsigned int bits = static_cast<unsigned int>(data[21]) |
                                (static_cast<unsigned int>(data[22]) << 8) |
                                (static_cast<unsigned int>(data[23]) << 16) |
                                (static_cast<unsigned int>(data[24]) << 24);
            width = static_cast<int>((bits & 0x3FFF) + 1);
            height = static_cast<int>(((bits >> 14) & 0x3FFF) + 1);
        } else {
            return false;
        }

        return width > 0 && height > 0;
    }
    
    return false;
}

// ---------------------------------------------------------------------------
// PNG codec
//
// A logo pipeline needs to decode a real image, scale it, and re-encode it.
// These helpers implement the subset of the PNG specification that logos use
// (8-bit truecolour and truecolour-with-alpha, single IDAT stream, no
// interlacing) directly on top of zlib, so no external image library is
// required and nothing is approximated.
// ---------------------------------------------------------------------------

namespace {

uint32_t read_be32(const unsigned char* p) {
    return (static_cast<uint32_t>(p[0]) << 24) | (static_cast<uint32_t>(p[1]) << 16) |
           (static_cast<uint32_t>(p[2]) << 8) | static_cast<uint32_t>(p[3]);
}

void write_be32(std::vector<unsigned char>& out, uint32_t v) {
    out.push_back(static_cast<unsigned char>((v >> 24) & 0xFF));
    out.push_back(static_cast<unsigned char>((v >> 16) & 0xFF));
    out.push_back(static_cast<unsigned char>((v >> 8) & 0xFF));
    out.push_back(static_cast<unsigned char>(v & 0xFF));
}

/// PNG filters operate on bytes, so undo them before inflating.
int paeth_predictor(int a, int b, int c) {
    int p = a + b - c;
    int pa = std::abs(p - a);
    int pb = std::abs(p - b);
    int pc = std::abs(p - c);
    if (pa <= pb && pa <= pc) return a;
    if (pb <= pc) return b;
    return c;
}

/// Decoded image: 8-bit RGBA, tightly packed, top row first.
struct DecodedImage {
    uint32_t width = 0;
    uint32_t height = 0;
    std::vector<unsigned char> rgba;  // width * height * 4
};

/// Decode an 8-bit PNG into RGBA. Returns false for a format this codec does
/// not implement, so a caller can refuse the upload instead of storing
/// something it cannot process.
bool decode_png(const std::vector<unsigned char>& data, DecodedImage& out) {
    static const unsigned char kSignature[8] = {0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A};
    if (data.size() < 8 || std::memcmp(data.data(), kSignature, 8) != 0) {
        return false;
    }

    uint32_t width = 0, height = 0;
    int bit_depth = 0, colour_type = 0;
    bool seen_ihdr = false;
    std::vector<unsigned char> idat;

    size_t pos = 8;
    while (pos + 8 <= data.size()) {
        uint32_t length = read_be32(&data[pos]);
        std::string type(reinterpret_cast<const char*>(&data[pos + 4]), 4);
        size_t body = pos + 8;
        if (body + length + 4 > data.size()) {
            return false;  // truncated chunk
        }

        if (type == "IHDR") {
            if (length < 13) return false;
            width = read_be32(&data[body]);
            height = read_be32(&data[body + 4]);
            bit_depth = data[body + 8];
            colour_type = data[body + 9];
            if (data[body + 12] != 0) return false;  // interlaced
            seen_ihdr = true;
        } else if (type == "IDAT") {
            idat.insert(idat.end(), data.begin() + body, data.begin() + body + length);
        } else if (type == "IEND") {
            break;
        }

        pos = body + length + 4;
    }

    if (!seen_ihdr || width == 0 || height == 0 || idat.empty()) {
        return false;
    }
    if (bit_depth != 8 || (colour_type != 2 && colour_type != 6)) {
        return false;  // only 8-bit RGB / RGBA
    }

    const int channels = (colour_type == 6) ? 4 : 3;
    const size_t stride = static_cast<size_t>(width) * channels;
    const size_t raw_size = (stride + 1) * height;

    std::vector<unsigned char> raw(raw_size);
    uLongf dest_len = static_cast<uLongf>(raw_size);
    if (uncompress(raw.data(), &dest_len, idat.data(),
                   static_cast<uLong>(idat.size())) != Z_OK) {
        return false;
    }
    if (dest_len < raw_size) {
        return false;
    }

    out.width = width;
    out.height = height;
    out.rgba.assign(static_cast<size_t>(width) * height * 4, 0);

    for (uint32_t y = 0; y < height; ++y) {
        const unsigned char filter = raw[y * (stride + 1)];
        const unsigned char* src = &raw[y * (stride + 1) + 1];
        unsigned char* dst = &out.rgba[static_cast<size_t>(y) * width * 4];
        const unsigned char* prev =
            (y == 0) ? nullptr : &out.rgba[static_cast<size_t>(y - 1) * width * 4];

        for (size_t x = 0; x < stride; ++x) {
            int a = (x >= static_cast<size_t>(channels))
                        ? dst[x - channels]
                        : 0;
            int b = prev ? prev[x] : 0;
            int c = (prev && x >= static_cast<size_t>(channels))
                        ? prev[x - channels]
                        : 0;
            int value = src[x];

            switch (filter) {
                case 0: break;                       // None
                case 1: value += a; break;           // Sub
                case 2: value += b; break;           // Up
                case 3: value += (a + b) / 2; break; // Average
                case 4: value += paeth_predictor(a, b, c); break;
                default: return false;
            }
            dst[x] = static_cast<unsigned char>(value & 0xFF);
        }

        // Widen RGB to RGBA so downstream code has one layout.
        if (channels == 3) {
            for (size_t x = width; x-- > 0;) {
                dst[x * 4 + 0] = dst[x * 3 + 0];
                dst[x * 4 + 1] = dst[x * 3 + 1];
                dst[x * 4 + 2] = dst[x * 3 + 2];
                dst[x * 4 + 3] = 255;
            }
        }
    }

    return true;
}

/// Encode RGBA as an 8-bit truecolour-with-alpha PNG (filter 0, single IDAT).
bool encode_png(const DecodedImage& img, std::vector<unsigned char>& out) {
    if (img.width == 0 || img.height == 0 ||
        img.rgba.size() != static_cast<size_t>(img.width) * img.height * 4) {
        return false;
    }

    const size_t stride = static_cast<size_t>(img.width) * 4;
    std::vector<unsigned char> raw((stride + 1) * img.height);
    for (uint32_t y = 0; y < img.height; ++y) {
        raw[y * (stride + 1)] = 0;  // filter: None
        std::memcpy(&raw[y * (stride + 1) + 1], &img.rgba[static_cast<size_t>(y) * stride], stride);
    }

    uLongf compressed_len = compressBound(static_cast<uLong>(raw.size()));
    std::vector<unsigned char> compressed(compressed_len);
    if (compress2(compressed.data(), &compressed_len, raw.data(),
                  static_cast<uLong>(raw.size()), Z_BEST_COMPRESSION) != Z_OK) {
        return false;
    }
    compressed.resize(compressed_len);

    static const unsigned char kSignature[8] = {0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A};
    out.assign(kSignature, kSignature + 8);

    auto append_chunk = [&out](const char* type, const std::vector<unsigned char>& body) {
        write_be32(out, static_cast<uint32_t>(body.size()));
        const size_t crc_start = out.size();
        out.insert(out.end(), type, type + 4);
        out.insert(out.end(), body.begin(), body.end());
        uLong crc = crc32(0L, Z_NULL, 0);
        crc = crc32(crc, &out[crc_start], static_cast<uInt>(4 + body.size()));
        write_be32(out, static_cast<uint32_t>(crc));
    };

    std::vector<unsigned char> ihdr;
    write_be32(ihdr, img.width);
    write_be32(ihdr, img.height);
    ihdr.push_back(8);  // bit depth
    ihdr.push_back(6);  // colour type: RGBA
    ihdr.push_back(0);  // compression: deflate
    ihdr.push_back(0);  // filter method
    ihdr.push_back(0);  // interlace: none
    append_chunk("IHDR", ihdr);
    append_chunk("IDAT", compressed);
    append_chunk("IEND", {});

    return true;
}

/// Area-average resample. Each destination pixel is the mean of the source
/// pixels it covers, which avoids the aliasing a nearest-neighbour scale
/// would introduce on the small icon sizes this service generates.
bool resample_rgba(const DecodedImage& src, uint32_t dst_w, uint32_t dst_h, DecodedImage& out) {
    if (src.width == 0 || src.height == 0 || dst_w == 0 || dst_h == 0) {
        return false;
    }

    out.width = dst_w;
    out.height = dst_h;
    out.rgba.assign(static_cast<size_t>(dst_w) * dst_h * 4, 0);

    for (uint32_t dy = 0; dy < dst_h; ++dy) {
        const uint64_t sy0 = static_cast<uint64_t>(dy) * src.height / dst_h;
        uint64_t sy1 = static_cast<uint64_t>(dy + 1) * src.height / dst_h;
        if (sy1 <= sy0) sy1 = sy0 + 1;
        if (sy1 > src.height) sy1 = src.height;

        for (uint32_t dx = 0; dx < dst_w; ++dx) {
            const uint64_t sx0 = static_cast<uint64_t>(dx) * src.width / dst_w;
            uint64_t sx1 = static_cast<uint64_t>(dx + 1) * src.width / dst_w;
            if (sx1 <= sx0) sx1 = sx0 + 1;
            if (sx1 > src.width) sx1 = src.width;

            uint64_t sums[4] = {0, 0, 0, 0};
            uint64_t count = 0;

            for (uint64_t sy = sy0; sy < sy1; ++sy) {
                for (uint64_t sx = sx0; sx < sx1; ++sx) {
                    const size_t idx = (static_cast<size_t>(sy) * src.width + sx) * 4;
                    // Weight colour by alpha so transparent pixels do not
                    // darken the edge of the scaled image.
                    const uint64_t alpha = src.rgba[idx + 3];
                    sums[0] += static_cast<uint64_t>(src.rgba[idx + 0]) * alpha;
                    sums[1] += static_cast<uint64_t>(src.rgba[idx + 1]) * alpha;
                    sums[2] += static_cast<uint64_t>(src.rgba[idx + 2]) * alpha;
                    sums[3] += alpha;
                    ++count;
                }
            }

            if (count == 0) {
                continue;
            }

            const size_t didx = (static_cast<size_t>(dy) * dst_w + dx) * 4;
            const uint64_t alpha_sum = sums[3];
            if (alpha_sum == 0) {
                out.rgba[didx + 0] = 0;
                out.rgba[didx + 1] = 0;
                out.rgba[didx + 2] = 0;
                out.rgba[didx + 3] = 0;
            } else {
                out.rgba[didx + 0] = static_cast<unsigned char>(sums[0] / alpha_sum);
                out.rgba[didx + 1] = static_cast<unsigned char>(sums[1] / alpha_sum);
                out.rgba[didx + 2] = static_cast<unsigned char>(sums[2] / alpha_sum);
                out.rgba[didx + 3] = static_cast<unsigned char>(alpha_sum / count);
            }
        }
    }

    return true;
}

}  // namespace

std::vector<unsigned char> ImageProcessor::createThumbnail(
    const std::vector<unsigned char>& image_data,
    int target_width,
    int target_height
) {
    if (target_width <= 0 || target_height <= 0) {
        return {};
    }

    DecodedImage source;
    if (!decode_png(image_data, source)) {
        return {};
    }

    DecodedImage scaled;
    if (!resample_rgba(source, static_cast<uint32_t>(target_width),
                       static_cast<uint32_t>(target_height), scaled)) {
        return {};
    }

    std::vector<unsigned char> out;
    if (!encode_png(scaled, out)) {
        return {};
    }
    return out;
}

std::vector<unsigned char> ImageProcessor::convertToPNG(const std::vector<unsigned char>& data) {
    DecodedImage image;
    if (!decode_png(data, image)) {
        return {};
    }

    std::vector<unsigned char> out;
    if (!encode_png(image, out)) {
        return {};
    }
    return out;
}

/// Re-encode losslessly at the highest deflate level. PNG is already lossless,
/// so the only honest "optimisation" available without a quantiser is a
/// stronger compression pass; the pixels are unchanged.
std::vector<unsigned char> ImageProcessor::optimizePNG(const std::vector<unsigned char>& data) {
    DecodedImage image;
    if (!decode_png(data, image)) {
        return {};
    }

    std::vector<unsigned char> out;
    if (!encode_png(image, out)) {
        return {};
    }

    // Keep the original if re-encoding did not actually shrink it.
    if (out.size() >= data.size()) {
        return data;
    }
    return out;
}

bool ImageProcessor::resizeImage(
    const std::vector<unsigned char>& input,
    std::vector<unsigned char>& output,
    int target_width,
    int target_height
) {
    output = createThumbnail(input, target_width, target_height);
    return !output.empty();
}

// ============================================================================
// Local Storage Backend Implementation
// ============================================================================

bool LocalStorageBackend::initialize(const LogoConfig& config) {
    config_ = config;

    std::error_code ec;
    std::filesystem::create_directories(config_.storage_path, ec);
    if (ec) {
        return false;
    }

    return std::filesystem::is_directory(config_.storage_path);
}

/// Reject a storage path that could escape the configured root. Paths reach
/// this backend from upload requests, so `..` segments and absolute paths
/// must not be able to place a file outside `storage_path`.
static bool resolve_within_root(
    const std::string& root,
    const std::string& path,
    std::string& out
) {
    if (path.empty() || path.front() == '/') {
        return false;
    }

    std::filesystem::path root_abs = std::filesystem::weakly_canonical(root);
    std::filesystem::path candidate = std::filesystem::weakly_canonical(root_abs / path);

    // Component-wise prefix check, so "/var/logos2" is not treated as being
    // inside "/var/logos".
    auto root_it = root_abs.begin();
    auto candidate_it = candidate.begin();
    for (; root_it != root_abs.end(); ++root_it, ++candidate_it) {
        if (candidate_it == candidate.end() || *root_it != *candidate_it) {
            return false;
        }
    }

    out = candidate.string();
    return true;
}

bool LocalStorageBackend::store(const std::string& path, const std::vector<unsigned char>& data) {
    std::lock_guard<std::mutex> lock(mutex_);

    std::string full_path;
    if (!resolve_within_root(config_.storage_path, path, full_path)) {
        return false;
    }

    std::error_code ec;
    std::filesystem::create_directories(
        std::filesystem::path(full_path).parent_path(), ec);
    if (ec) {
        return false;
    }

    std::ofstream file(full_path, std::ios::binary | std::ios::trunc);
    if (!file.is_open()) {
        return false;
    }

    file.write(reinterpret_cast<const char*>(data.data()),
               static_cast<std::streamsize>(data.size()));
    file.close();

    return file.good();
}

std::vector<unsigned char> LocalStorageBackend::retrieve(const std::string& path) {
    std::lock_guard<std::mutex> lock(mutex_);

    std::string full_path;
    if (!resolve_within_root(config_.storage_path, path, full_path)) {
        return {};
    }

    std::ifstream file(full_path, std::ios::binary | std::ios::ate);

    if (!file.is_open()) {
        return {};
    }

    std::streamsize size = file.tellg();
    file.seekg(0, std::ios::beg);
    
    std::vector<unsigned char> buffer(size);
    if (!file.read(reinterpret_cast<char*>(buffer.data()), size)) {
        return {};
    }
    
    return buffer;
}

bool LocalStorageBackend::remove(const std::string& path) {
    std::lock_guard<std::mutex> lock(mutex_);

    std::string full_path;
    if (!resolve_within_root(config_.storage_path, path, full_path)) {
        return false;
    }

    std::error_code ec;
    std::filesystem::remove(full_path, ec);
    return !ec;
}

std::string LocalStorageBackend::getURL(const std::string& path) {
    return config_.cdn_base_url + "/" + path;
}

// ============================================================================
// S3 Storage Backend Implementation
// ============================================================================

S3StorageBackend::S3StorageBackend() : curl_(nullptr), curl_mutex_() {
    curl_ = curl_easy_init();
}

S3StorageBackend::~S3StorageBackend() {
    if (curl_) {
        curl_easy_cleanup(curl_);
    }
}

namespace {

std::string hex_encode(const unsigned char* data, size_t len) {
    static const char* kHex = "0123456789abcdef";
    std::string out;
    out.reserve(len * 2);
    for (size_t i = 0; i < len; ++i) {
        out.push_back(kHex[data[i] >> 4]);
        out.push_back(kHex[data[i] & 0x0F]);
    }
    return out;
}

std::string sha256_hex(const std::string& data) {
    unsigned char digest[SHA256_DIGEST_LENGTH];
    SHA256(reinterpret_cast<const unsigned char*>(data.data()), data.size(), digest);
    return hex_encode(digest, SHA256_DIGEST_LENGTH);
}

std::vector<unsigned char> hmac_sha256(
    const std::vector<unsigned char>& key,
    const std::string& data
) {
    unsigned int len = EVP_MAX_MD_SIZE;
    std::vector<unsigned char> out(EVP_MAX_MD_SIZE);
    HMAC(EVP_sha256(), key.data(), static_cast<int>(key.size()),
         reinterpret_cast<const unsigned char*>(data.data()), data.size(),
         out.data(), &len);
    out.resize(len);
    return out;
}

/// RFC 3986 encoding as AWS SigV4 requires: unreserved characters pass
/// through, everything else becomes %XX. A '/' is preserved when encoding a
/// path but not when encoding a query value.
std::string uri_encode(const std::string& value, bool encode_slash) {
    static const char* kHex = "0123456789ABCDEF";
    std::string out;
    for (unsigned char c : value) {
        const bool unreserved = (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
                                (c >= '0' && c <= '9') || c == '-' || c == '_' ||
                                c == '.' || c == '~';
        if (unreserved || (c == '/' && !encode_slash)) {
            out.push_back(static_cast<char>(c));
        } else {
            out.push_back('%');
            out.push_back(kHex[c >> 4]);
            out.push_back(kHex[c & 0x0F]);
        }
    }
    return out;
}

/// Current UTC time as the "YYYYMMDDTHHMMSSZ" and "YYYYMMDD" forms SigV4 needs.
void sigv4_timestamps(std::string& amz_date, std::string& date_stamp) {
    std::time_t now = std::time(nullptr);
    std::tm utc{};
    gmtime_r(&now, &utc);

    char full[32];
    char short_form[16];
    std::strftime(full, sizeof(full), "%Y%m%dT%H%M%SZ", &utc);
    std::strftime(short_form, sizeof(short_form), "%Y%m%d", &utc);
    amz_date = full;
    date_stamp = short_form;
}

}  // namespace

bool S3StorageBackend::initialize(const LogoConfig& config) {
    config_ = config;
    bucket_ = config.aws_bucket;
    region_ = config.aws_region;

    if (!curl_) {
        return false;
    }

    // Credentials are mandatory: an unsigned request is rejected by S3, so
    // accepting the configuration would only defer the failure.
    if (bucket_.empty() || region_.empty() ||
        config_.aws_access_key.empty() || config_.aws_secret_key.empty()) {
        return false;
    }

    return true;
}

/// Sign a request with AWS Signature Version 4. The canonical request and the
/// derived signing key follow the algorithm in the AWS documentation.
/// Derive the SigV4 signing key. Per the specification the first HMAC is keyed
/// with "AWS4" + dateStamp over the dateStamp itself; keying with the raw
/// secret instead produces signatures that S3 rejects with
/// SignatureDoesNotMatch.
std::vector<unsigned char> S3StorageBackend::derive_signing_key(
    const std::string& date_stamp,
    const std::string& region
) {
    const std::string prefix = "AWS4" + date_stamp;
    std::vector<unsigned char> key(prefix.begin(), prefix.end());

    std::vector<unsigned char> k_date = hmac_sha256(key, date_stamp);
    std::vector<unsigned char> k_region = hmac_sha256(k_date, region);
    std::vector<unsigned char> k_service = hmac_sha256(k_region, "s3");
    return hmac_sha256(k_service, "aws4_request");
}

bool S3StorageBackend::signRequest(
    const std::string& method,
    const std::string& canonical_uri,
    const std::string& canonical_query,
    const std::string& payload,
    const std::string& content_type,
    std::vector<std::string>& headers
) {
    std::string amz_date;
    std::string date_stamp;
    sigv4_timestamps(amz_date, date_stamp);

    const std::string host = bucket_ + ".s3." + region_ + ".amazonaws.com";
    const std::string payload_hash = sha256_hex(payload);

    // Canonical headers must be sorted by lowercase name.
    std::string canonical_headers =
        "content-type:" + content_type + "\n" +
        "host:" + host + "\n" +
        "x-amz-content-sha256:" + payload_hash + "\n" +
        "x-amz-date:" + amz_date + "\n";
    const std::string signed_headers = "content-type;host;x-amz-content-sha256;x-amz-date";

    const std::string canonical_request =
        method + "\n" + canonical_uri + "\n" + canonical_query + "\n" +
        canonical_headers + "\n" + signed_headers + "\n" + payload_hash;

    const std::string scope =
        date_stamp + "/" + region_ + "/s3/aws4_request";
    const std::string string_to_sign =
        "AWS4-HMAC-SHA256\n" + amz_date + "\n" + scope + "\n" +
        sha256_hex(canonical_request);

    std::vector<unsigned char> k_signing = derive_signing_key(date_stamp, region_);
    std::vector<unsigned char> signature_bytes = hmac_sha256(k_signing, string_to_sign);

    const std::string authorization =
        "AWS4-HMAC-SHA256 Credential=" + config_.aws_access_key + "/" + scope +
        ", SignedHeaders=" + signed_headers +
        ", Signature=" + hex_encode(signature_bytes.data(), signature_bytes.size());

    headers.push_back("Content-Type: " + content_type);
    headers.push_back("x-amz-content-sha256: " + payload_hash);
    headers.push_back("x-amz-date: " + amz_date);
    headers.push_back("Authorization: " + authorization);
    return true;
}

bool S3StorageBackend::store(const std::string& path, const std::vector<unsigned char>& data) {
    std::lock_guard<std::mutex> lock(curl_mutex_);

    if (!curl_ || path.empty() || path.front() == '/') {
        return false;
    }

    return uploadToS3(path, data);
}

bool S3StorageBackend::uploadToS3(const std::string& path, const std::vector<unsigned char>& data) {
    if (!curl_) return false;

    const std::string canonical_uri = "/" + uri_encode(path, false);
    const std::string url =
        "https://" + bucket_ + ".s3." + region_ + ".amazonaws.com" + canonical_uri;
    const std::string payload(reinterpret_cast<const char*>(data.data()), data.size());

    std::vector<std::string> header_strings;
    if (!signRequest("PUT", canonical_uri, "", payload, "image/png", header_strings)) {
        return false;
    }

    struct curl_slist* headers = nullptr;
    for (const auto& h : header_strings) {
        headers = curl_slist_append(headers, h.c_str());
    }

    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl_, CURLOPT_CUSTOMREQUEST, "PUT");
    curl_easy_setopt(curl_, CURLOPT_POSTFIELDSIZE, static_cast<long>(data.size()));
    curl_easy_setopt(curl_, CURLOPT_POSTFIELDS, data.data());
    curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers);

    CURLcode res = curl_easy_perform(curl_);

    long status = 0;
    curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &status);

    curl_slist_free_all(headers);

    return res == CURLE_OK && status >= 200 && status < 300;
}

std::vector<unsigned char> S3StorageBackend::retrieve(const std::string& path) {
    std::lock_guard<std::mutex> lock(curl_mutex_);

    if (!curl_ || path.empty() || path.front() == '/') {
        return {};
    }

    const std::string canonical_uri = "/" + uri_encode(path, false);
    const std::string url =
        "https://" + bucket_ + ".s3." + region_ + ".amazonaws.com" + canonical_uri;

    std::vector<std::string> header_strings;
    if (!signRequest("GET", canonical_uri, "", "", "application/octet-stream", header_strings)) {
        return {};
    }

    struct curl_slist* headers = nullptr;
    for (const auto& h : header_strings) {
        headers = curl_slist_append(headers, h.c_str());
    }

    std::string response;
    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl_, CURLOPT_HTTPGET, 1L);
    curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION,
        [](void* contents, size_t size, size_t nmemb, void* userp) {
            ((std::string*)userp)->append((char*)contents, size * nmemb);
            return size * nmemb;
        });
    curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &response);

    CURLcode res = curl_easy_perform(curl_);

    long status = 0;
    curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &status);

    curl_slist_free_all(headers);

    if (res != CURLE_OK || status < 200 || status >= 300) {
        return {};
    }

    return std::vector<unsigned char>(response.begin(), response.end());
}

bool S3StorageBackend::remove(const std::string& path) {
    std::lock_guard<std::mutex> lock(curl_mutex_);

    if (!curl_ || path.empty() || path.front() == '/') {
        return false;
    }

    const std::string canonical_uri = "/" + uri_encode(path, false);
    const std::string url =
        "https://" + bucket_ + ".s3." + region_ + ".amazonaws.com" + canonical_uri;

    std::vector<std::string> header_strings;
    if (!signRequest("DELETE", canonical_uri, "", "", "application/octet-stream", header_strings)) {
        return false;
    }

    struct curl_slist* headers = nullptr;
    for (const auto& h : header_strings) {
        headers = curl_slist_append(headers, h.c_str());
    }

    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl_, CURLOPT_CUSTOMREQUEST, "DELETE");
    curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl_, CURLOPT_NOBODY, 0L);

    CURLcode res = curl_easy_perform(curl_);

    long status = 0;
    curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &status);

    curl_slist_free_all(headers);

    // S3 returns 204 for a successful delete and 404 when the object is
    // already gone; both mean the object is not present afterwards.
    return res == CURLE_OK && (status == 204 || status == 200 || status == 404);
}

std::string S3StorageBackend::getURL(const std::string& path) {
    return "https://" + bucket_ + ".s3." + region_ + ".amazonaws.com/" +
           uri_encode(path, false);
}

/// Build a SigV4 query-string presigned URL, which is the form a browser can
/// use directly without holding credentials.
std::string S3StorageBackend::generatePresignedURL(const std::string& path) {
    if (bucket_.empty() || region_.empty() ||
        config_.aws_access_key.empty() || config_.aws_secret_key.empty()) {
        return "";
    }

    std::string amz_date;
    std::string date_stamp;
    sigv4_timestamps(amz_date, date_stamp);

    const std::string host = bucket_ + ".s3." + region_ + ".amazonaws.com";
    const std::string canonical_uri = "/" + uri_encode(path, false);
    const std::string scope = date_stamp + "/" + region_ + "/s3/aws4_request";
    const std::string credential =
        uri_encode(config_.aws_access_key + "/" + scope, true);

    // Query parameters must be sorted by name for the canonical request.
    const std::string canonical_query =
        "X-Amz-Algorithm=AWS4-HMAC-SHA256"
        "&X-Amz-Credential=" + credential +
        "&X-Amz-Date=" + amz_date +
        "&X-Amz-Expires=3600"
        "&X-Amz-SignedHeaders=host";

    const std::string canonical_headers = "host:" + host + "\n";
    const std::string payload_hash = sha256_hex("");

    const std::string canonical_request =
        "GET\n" + canonical_uri + "\n" + canonical_query + "\n" +
        canonical_headers + "\n" + "host" + "\n" + payload_hash;

    const std::string string_to_sign =
        "AWS4-HMAC-SHA256\n" + amz_date + "\n" + scope + "\n" +
        sha256_hex(canonical_request);

    std::vector<unsigned char> k_signing = derive_signing_key(date_stamp, region_);
    std::vector<unsigned char> signature_bytes = hmac_sha256(k_signing, string_to_sign);

    return "https://" + host + canonical_uri + "?" + canonical_query +
           "&X-Amz-Signature=" + hex_encode(signature_bytes.data(), signature_bytes.size());
}

// ============================================================================
// CDN Manager Implementation
// ============================================================================

CDNManager::CDNManager() {}

CDNManager::~CDNManager() {}

bool CDNManager::initialize(const LogoConfig& config) {
    config_ = config;
    
    // Use local storage by default
    storage_ = std::make_unique<LocalStorageBackend>();
    
    LogoConfig storage_config = config;
    if (config_.enable_cdn && !config_.aws_bucket.empty()) {
        storage_ = std::make_unique<S3StorageBackend>();
    }
    
    return storage_->initialize(config);
}

CDNUploadResult CDNManager::upload(
    const std::string& path,
    const std::vector<unsigned char>& data,
    const std::string& content_type
) {
    CDNUploadResult result;
    auto start = std::chrono::high_resolution_clock::now();
    
    // Store to storage backend
    bool success = storage_->store(path, data);
    
    auto end = std::chrono::high_resolution_clock::now();
    result.upload_time_ms = std::chrono::duration_cast<std::chrono::milliseconds>(end - start).count();
    
    if (success) {
        result.success = true;
        result.cdn_url = storage_->getURL(path);
        
        // Cache the result
        std::lock_guard<std::mutex> lock(cache_mutex_);
        cache_[generateCacheKey(path)] = result.cdn_url;
    } else {
        result.success = false;
        result.error_message = "Failed to upload to storage";
    }
    
    return result;
}

bool CDNManager::invalidateCache(const std::string& path) {
    std::lock_guard<std::mutex> lock(cache_mutex_);
    cache_.erase(generateCacheKey(path));
    return true;
}

std::string CDNManager::getCDNURL(const std::string& path) {
    std::lock_guard<std::mutex> lock(cache_mutex_);
    
    auto it = cache_.find(generateCacheKey(path));
    if (it != cache_.end()) {
        return it->second;
    }
    
    return storage_->getURL(path);
}

std::string CDNManager::generateCacheKey(const std::string& path) {
    return path;
}

// ============================================================================
// Redis Cache Implementation
// ============================================================================

RedisCache::RedisCache() : socket_fd_(-1), connected_(false) {}

RedisCache::~RedisCache() {
    if (socket_fd_ >= 0) {
        ::close(socket_fd_);
        socket_fd_ = -1;
    }
    connected_ = false;
}

bool RedisCache::connect(const std::string& host, int port) {
    std::lock_guard<std::mutex> lock(mutex_);

    if (socket_fd_ >= 0) {
        ::close(socket_fd_);
        socket_fd_ = -1;
    }
    connected_ = false;

    config_.redis_host = host;
    config_.redis_port = port;

    struct addrinfo hints;
    std::memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;

    struct addrinfo* result = nullptr;
    std::string port_str = std::to_string(port);
    if (::getaddrinfo(host.c_str(), port_str.c_str(), &hints, &result) != 0) {
        return false;
    }

    int fd = -1;
    for (struct addrinfo* rp = result; rp != nullptr; rp = rp->ai_next) {
        fd = ::socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
        if (fd < 0) {
            continue;
        }
        if (::connect(fd, rp->ai_addr, rp->ai_addrlen) == 0) {
            break;
        }
        ::close(fd);
        fd = -1;
    }
    ::freeaddrinfo(result);

    if (fd < 0) {
        return false;
    }

    struct timeval timeout;
    timeout.tv_sec = 5;
    timeout.tv_usec = 0;
    ::setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
    ::setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));

    socket_fd_ = fd;
    connected_ = true;
    return true;
}

/// Serialize arguments as a RESP array of bulk strings. RESP is binary safe,
/// so a value containing CRLF or NUL is transmitted verbatim rather than
/// being able to inject an additional command.
std::string RedisCache::buildRedisCommand(const std::vector<std::string>& args) {
    std::string out = "*" + std::to_string(args.size()) + "\r\n";
    for (const auto& arg : args) {
        out += "$" + std::to_string(arg.size()) + "\r\n";
        out += arg;
        out += "\r\n";
    }
    return out;
}

static bool write_all(int fd, const std::string& data) {
    size_t written = 0;
    while (written < data.size()) {
        ssize_t n = ::send(fd, data.data() + written, data.size() - written, 0);
        if (n <= 0) {
            return false;
        }
        written += static_cast<size_t>(n);
    }
    return true;
}

/// Read exactly `n` bytes, or false if the peer closes or times out first.
static bool read_exact(int fd, char* buf, size_t n) {
    size_t got = 0;
    while (got < n) {
        ssize_t r = ::recv(fd, buf + got, n - got, 0);
        if (r <= 0) {
            return false;
        }
        got += static_cast<size_t>(r);
    }
    return true;
}

static bool read_line(int fd, std::string& line) {
    line.clear();
    char c = 0;
    while (true) {
        if (!read_exact(fd, &c, 1)) {
            return false;
        }
        if (c == '\r') {
            char lf = 0;
            if (!read_exact(fd, &lf, 1) || lf != '\n') {
                return false;
            }
            return true;
        }
        line.push_back(c);
        if (line.size() > 64 * 1024) {
            return false;
        }
    }
}

bool RedisCache::sendCommand(const std::vector<std::string>& args) {
    std::lock_guard<std::mutex> lock(mutex_);

    if (!connected_ || socket_fd_ < 0) {
        return false;
    }

    if (!write_all(socket_fd_, buildRedisCommand(args))) {
        connected_ = false;
        return false;
    }

    std::string reply;
    if (!readReply(reply)) {
        connected_ = false;
        return false;
    }

    return true;
}

/// Parse one RESP reply. Returns false on a transport error or a `-ERR`
/// reply, so an error is never surfaced to a caller as a value.
bool RedisCache::readReply(std::string& out) {
    char type = 0;
    if (!read_exact(socket_fd_, &type, 1)) {
        return false;
    }

    std::string line;
    switch (type) {
        case '+':  // simple string
            if (!read_line(socket_fd_, line)) return false;
            out = line;
            return true;

        case '-':  // error
            if (!read_line(socket_fd_, line)) return false;
            out = line;
            return false;

        case ':':  // integer
            if (!read_line(socket_fd_, line)) return false;
            out = line;
            return true;

        case '$': {  // bulk string
            if (!read_line(socket_fd_, line)) return false;
            long long len = 0;
            try {
                len = std::stoll(line);
            } catch (...) {
                return false;
            }
            if (len < 0) {
                out.clear();  // RESP nil
                return true;
            }
            std::string buf(static_cast<size_t>(len), '\0');
            if (!read_exact(socket_fd_, &buf[0], static_cast<size_t>(len))) {
                return false;
            }
            char crlf[2];
            if (!read_exact(socket_fd_, crlf, 2)) {
                return false;
            }
            out = buf;
            return true;
        }

        default:
            // Arrays and other aggregate types are not used by this client.
            return false;
    }
}

bool RedisCache::set(const std::string& key, const std::string& value, int ttl_seconds) {
    if (ttl_seconds > 0) {
        return sendCommand({"SET", key, value, "EX", std::to_string(ttl_seconds)});
    }
    return sendCommand({"SET", key, value});
}

std::string RedisCache::get(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);

    if (!connected_ || socket_fd_ < 0) {
        return "";
    }

    if (!write_all(socket_fd_, buildRedisCommand({"GET", key}))) {
        connected_ = false;
        return "";
    }

    std::string reply;
    if (!readReply(reply)) {
        connected_ = false;
        return "";
    }
    return reply;
}

bool RedisCache::del(const std::string& key) {
    return sendCommand({"DEL", key});
}

bool RedisCache::exists(const std::string& key) {
    std::lock_guard<std::mutex> lock(mutex_);

    if (!connected_ || socket_fd_ < 0) {
        return false;
    }

    if (!write_all(socket_fd_, buildRedisCommand({"EXISTS", key}))) {
        connected_ = false;
        return false;
    }

    std::string reply;
    if (!readReply(reply)) {
        connected_ = false;
        return false;
    }

    try {
        return std::stoll(reply) > 0;
    } catch (...) {
        return false;
    }
}

bool RedisCache::ping() {
    std::lock_guard<std::mutex> lock(mutex_);

    if (!connected_ || socket_fd_ < 0) {
        return false;
    }

    if (!write_all(socket_fd_, buildRedisCommand({"PING"}))) {
        connected_ = false;
        return false;
    }

    std::string reply;
    if (!readReply(reply)) {
        connected_ = false;
        return false;
    }

    return reply == "PONG";
}

// ============================================================================
// Logo Upload Service Implementation
// ============================================================================

LogoUploadService::LogoUploadService()
    : running_(false)
    , total_processed_(0)
    , total_failed_(0) {
    start_time_ = std::chrono::steady_clock::now();
}

LogoUploadService::~LogoUploadService() {
    stop();
}

bool LogoUploadService::initialize(const LogoConfig& config) {
    config_ = config;
    
    // Initialize CDN manager
    cdn_manager_ = std::make_unique<CDNManager>();
    if (!cdn_manager_->initialize(config)) {
        std::cerr << "Failed to initialize CDN manager" << std::endl;
        return false;
    }
    
    // Initialize Redis cache
    redis_cache_ = std::make_unique<RedisCache>();
    if (config.enable_cache) {
        if (!redis_cache_->connect(config.redis_host, config.redis_port)) {
            std::cerr << "Warning: Failed to connect to Redis" << std::endl;
        }
    }
    
    return true;
}

void LogoUploadService::start() {
    if (running_) return;
    
    running_ = true;
    
    // Start worker threads
    for (int i = 0; i < config_.worker_threads; i++) {
        worker_threads_.emplace_back(&LogoUploadService::processQueue, this);
    }
    
    std::cout << "Logo upload service started with " << config_.worker_threads << " workers" << std::endl;
}

void LogoUploadService::stop() {
    if (!running_) return;
    
    running_ = false;
    queue_cv_.notify_all();
    
    for (auto& thread : worker_threads_) {
        if (thread.joinable()) {
            thread.join();
        }
    }
    
    worker_threads_.clear();
    std::cout << "Logo upload service stopped" << std::endl;
}

void LogoUploadService::upload(
    const std::string& token_symbol,
    const std::string& chain,
    const std::vector<unsigned char>& data,
    const std::string& original_filename,
    std::function<void(const LogoMetadata&, const std::string&)> callback
) {
    // Validate upload
    std::string error;
    if (!validateUpload(data, original_filename, error)) {
        if (callback) {
            LogoMetadata empty;
            callback(empty, error);
        }
        return;
    }
    
    // Create processing task
    auto task = std::make_shared<ProcessingTask>();
    task->task_id = generateLogoId(token_symbol, chain);
    task->request_id = task->task_id;
    task->original_data = data;
    task->original_filename = original_filename;
    task->token_symbol = token_symbol;
    task->chain = chain;
    task->start_time = std::chrono::steady_clock::now();
    task->callback = callback;
    
    // Queue the task
    {
        std::lock_guard<std::mutex> lock(queue_mutex_);
        
        if (task_queue_.size() >= (size_t)config_.max_queue_size) {
            if (callback) {
                LogoMetadata empty;
                callback(empty, "Queue full, please try again later");
            }
            return;
        }
        
        task_queue_.push(task);
    }
    
    queue_cv_.notify_one();
}

std::shared_ptr<ProcessingTask> LogoUploadService::dequeueTask() {
    std::unique_lock<std::mutex> lock(queue_mutex_);
    
    queue_cv_.wait_for(lock, std::chrono::seconds(1), [this] {
        return !task_queue_.empty() || !running_;
    });
    
    if (!running_ || task_queue_.empty()) {
        return nullptr;
    }
    
    auto task = task_queue_.front();
    task_queue_.pop();
    return task;
}

void LogoUploadService::processQueue() {
    while (running_) {
        auto task = dequeueTask();
        if (!task) continue;
        
        try {
            // Process the image
            auto metadata = processImage(
                task->original_data,
                task->original_filename,
                task->token_symbol,
                task->chain
            );
            
            if (metadata) {
                // Store original
                storeOriginal(metadata->id, task->original_data);
                
                // Generate thumbnails
                generateThumbnails(metadata->id, task->original_data);
                
                // Upload to CDN
                uploadToCDN(metadata->id);
                
                // Cache metadata
                cacheMetadata(metadata);
                
                total_processed_++;

                if (task->callback) {
                    task->callback(*metadata, "");
                }
            } else {
                total_failed_++;

                if (task->callback) {
                    LogoMetadata empty;
                    task->callback(empty, "image processing failed");
                }
            }
        } catch (const std::exception& e) {
            std::cerr << "Error processing task: " << e.what() << std::endl;
            total_failed_++;

            // A throw would otherwise leave the caller waiting forever.
            if (task->callback) {
                LogoMetadata empty;
                task->callback(empty, e.what());
            }
        }
    }
}

std::shared_ptr<LogoMetadata> LogoUploadService::processImage(
    const std::vector<unsigned char>& data,
    const std::string& original_filename,
    const std::string& token_symbol,
    const std::string& chain
) {
    auto metadata = std::make_shared<LogoMetadata>();
    
    metadata->id = generateLogoId(token_symbol, chain);
    metadata->original_filename = original_filename;
    metadata->content_hash = ImageProcessor::computeHash(data);
    metadata->token_symbol = token_symbol;
    metadata->chain = chain;
    metadata->created_at = getCurrentTimestamp();
    metadata->file_size = data.size();
    
    // Validate and get image info
    std::string mime_type;
    int width, height;
    if (!ImageProcessor::validateImage(data, mime_type, width, height)) {
        return nullptr;
    }
    
    metadata->mime_type = mime_type;
    metadata->width = width;
    metadata->height = height;
    
    // Set file extension based on format
    std::string format = ImageProcessor::getImageFormat(data);
    if (format == "png") {
        metadata->stored_filename = metadata->id + ".png";
    } else if (format == "jpeg") {
        metadata->stored_filename = metadata->id + ".jpg";
    } else {
        metadata->stored_filename = metadata->id + ".png";
    }
    
    return metadata;
}

bool LogoUploadService::storeOriginal(const std::string& logo_id, const std::vector<unsigned char>& data) {
    std::string path = "logos/" + logo_id + "/original";
    
    std::string format = ImageProcessor::getImageFormat(data);
    if (format == "png") {
        path += ".png";
    } else if (format == "jpeg") {
        path += ".jpg";
    }
    
    return cdn_manager_->upload(path, data, "image/png").success;
}

bool LogoUploadService::generateThumbnails(const std::string& logo_id, const std::vector<unsigned char>& data) {
    // Generate thumbnails at different sizes
    int sizes[] = {16, 32, 64, 128};
    
    for (int size : sizes) {
        auto thumbnail = ImageProcessor::createThumbnail(data, size, size);
        
        std::string path = "logos/" + logo_id + "/thumb_" + std::to_string(size);
        
        cdn_manager_->upload(path, thumbnail, "image/png");
    }
    
    return true;
}

bool LogoUploadService::uploadToCDN(const std::string& logo_id) {
    // Already done during storage
    return true;
}

void LogoUploadService::cacheMetadata(const std::shared_ptr<LogoMetadata>& metadata) {
    std::lock_guard<std::mutex> lock(metadata_mutex_);
    metadata_cache_[metadata->id] = metadata;
    
    // Also cache in Redis
    if (redis_cache_->ping()) {
        std::string key = "logo:metadata:" + metadata->id;
        // Serialize and cache
    }
}

std::shared_ptr<LogoMetadata> LogoUploadService::getCachedMetadata(const std::string& logo_id) {
    std::lock_guard<std::mutex> lock(metadata_mutex_);
    
    auto it = metadata_cache_.find(logo_id);
    if (it != metadata_cache_.end()) {
        return it->second;
    }
    
    return nullptr;
}

LogoMetadata LogoUploadService::getMetadata(const std::string& logo_id) {
    auto cached = getCachedMetadata(logo_id);
    if (cached) {
        return *cached;
    }
    
    LogoMetadata empty;
    return empty;
}

bool LogoUploadService::deleteLogo(const std::string& logo_id) {
    // Remove from cache
    {
        std::lock_guard<std::mutex> lock(metadata_mutex_);
        metadata_cache_.erase(logo_id);
    }
    
    // Invalidate CDN cache
    cdn_manager_->invalidateCache("logos/" + logo_id);
    
    return true;
}

std::vector<LogoMetadata> LogoUploadService::getLogosByToken(const std::string& token_symbol) {
    std::vector<LogoMetadata> result;
    
    std::lock_guard<std::mutex> lock(metadata_mutex_);
    
    for (const auto& pair : metadata_cache_) {
        if (pair.second->token_symbol == token_symbol) {
            result.push_back(*pair.second);
        }
    }
    
    return result;
}

LogoUploadService::ServiceStats LogoUploadService::getStats() {
    ServiceStats stats;
    stats.total_processed = total_processed_;
    stats.total_failed = total_failed_;
    
    auto now = std::chrono::steady_clock::now();
    auto elapsed = std::chrono::duration_cast<std::chrono::seconds>(now - start_time_).count();
    
    if (elapsed > 0) {
        stats.throughput_per_second = (double)total_processed_ / elapsed;
    } else {
        stats.throughput_per_second = 0;
    }
    
    stats.avg_processing_time_ms = 0; // Could track this
    stats.queue_size = task_queue_.size();
    stats.cache_size = metadata_cache_.size();
    
    return stats;
}

std::string LogoUploadService::generateLogoId(const std::string& token_symbol, const std::string& chain) {
    std::string input = token_symbol + "_" + chain + "_" + getCurrentTimestamp();
    std::string hash = ImageProcessor::computeHash(
        std::vector<unsigned char>(input.begin(), input.end())
    );
    return hash.substr(0, 16);
}

std::string LogoUploadService::getCurrentTimestamp() {
    auto now = std::chrono::system_clock::now();
    auto time_t_now = std::chrono::system_clock::to_time_t(now);
    
    std::stringstream ss;
    ss << std::put_time(std::localtime(&time_t_now), "%Y-%m-%dT%H:%M:%S");
    return ss.str();
}

bool LogoUploadService::validateUpload(
    const std::vector<unsigned char>& data,
    const std::string& filename,
    std::string& error
) {
    // Check file size
    if (data.size() > (size_t)config_.max_file_size_mb * 1024 * 1024) {
        error = "File too large. Maximum size: " + std::to_string(config_.max_file_size_mb) + "MB";
        return false;
    }
    
    // Validate image
    std::string mime_type;
    int width, height;
    if (!ImageProcessor::validateImage(data, mime_type, width, height)) {
        error = "Invalid image format. Allowed: PNG, JPEG, GIF, WebP";
        return false;
    }
    
    // Check dimensions
    if (width > 2048 || height > 2048) {
        error = "Image too large. Maximum dimensions: 2048x2048";
        return false;
    }
    
    if (width < 32 || height < 32) {
        error = "Image too small. Minimum dimensions: 32x32";
        return false;
    }
    
    return true;
}

// ============================================================================
// HTTP Server Implementation
// ============================================================================

LogoUploadHTTPServer::LogoUploadHTTPServer(LogoUploadService* service, int port)
    : service_(service), port_(port), running_(false) {}

LogoUploadHTTPServer::~LogoUploadHTTPServer() {
    stop();
}

void LogoUploadHTTPServer::start() {
    if (running_) return;
    
    running_ = true;
    server_thread_ = std::thread(&LogoUploadHTTPServer::runServer, this);
}

void LogoUploadHTTPServer::stop() {
    if (!running_) return;
    
    running_ = false;
    
    if (server_thread_.joinable()) {
        server_thread_.join();
    }
}

namespace {

/// Escape a string for embedding in a JSON document.
std::string json_escape(const std::string& value) {
    std::string out;
    out.reserve(value.size() + 8);
    for (unsigned char c : value) {
        switch (c) {
            case '"':  out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\n': out += "\\n"; break;
            case '\r': out += "\\r"; break;
            case '\t': out += "\\t"; break;
            default:
                if (c < 0x20) {
                    static const char* kHex = "0123456789abcdef";
                    out += "\\u00";
                    out.push_back(kHex[c >> 4]);
                    out.push_back(kHex[c & 0x0F]);
                } else {
                    out.push_back(static_cast<char>(c));
                }
        }
    }
    return out;
}

std::string metadata_to_json(const LogoMetadata& m) {
    return "{\"id\":\"" + json_escape(m.id) + "\""
           ",\"original_filename\":\"" + json_escape(m.original_filename) + "\""
           ",\"stored_filename\":\"" + json_escape(m.stored_filename) + "\""
           ",\"content_hash\":\"" + json_escape(m.content_hash) + "\""
           ",\"mime_type\":\"" + json_escape(m.mime_type) + "\""
           ",\"file_size\":" + std::to_string(m.file_size) +
           ",\"width\":" + std::to_string(m.width) +
           ",\"height\":" + std::to_string(m.height) +
           ",\"thumbnail_url\":\"" + json_escape(m.thumbnail_url) + "\""
           ",\"cdn_url\":\"" + json_escape(m.cdn_url) + "\""
           ",\"created_at\":\"" + json_escape(m.created_at) + "\""
           ",\"token_symbol\":\"" + json_escape(m.token_symbol) + "\""
           ",\"chain\":\"" + json_escape(m.chain) + "\"}";
}

struct MultipartBody {
    std::string file_data;
    std::string filename;
    std::map<std::string, std::string> fields;
};

std::string multipart_boundary(const std::string& content_type) {
    const std::string key = "boundary=";
    size_t pos = content_type.find(key);
    if (pos == std::string::npos) {
        return "";
    }
    std::string boundary = content_type.substr(pos + key.size());
    if (!boundary.empty() && boundary.front() == '"') {
        size_t end = boundary.find('"', 1);
        boundary = boundary.substr(1, end == std::string::npos ? std::string::npos : end - 1);
    }
    size_t semi = boundary.find(';');
    if (semi != std::string::npos) {
        boundary = boundary.substr(0, semi);
    }
    return boundary;
}

std::string trim_ws(const std::string& s) {
    size_t a = s.find_first_not_of(" \t\r\n");
    if (a == std::string::npos) return "";
    size_t b = s.find_last_not_of(" \t\r\n");
    return s.substr(a, b - a + 1);
}

bool parse_multipart(
    const std::string& body,
    const std::string& boundary,
    MultipartBody& out
) {
    if (boundary.empty()) {
        return false;
    }

    const std::string delimiter = "--" + boundary;
    size_t pos = body.find(delimiter);
    if (pos == std::string::npos) {
        return false;
    }
    pos += delimiter.size();

    while (pos < body.size()) {
        if (pos + 2 <= body.size() && body.compare(pos, 2, "--") == 0) {
            return true;
        }
        if (pos + 2 <= body.size() && body.compare(pos, 2, "\r\n") == 0) {
            pos += 2;
        }

        size_t header_end = body.find("\r\n\r\n", pos);
        if (header_end == std::string::npos) {
            return false;
        }

        const std::string raw_headers = body.substr(pos, header_end - pos);
        const size_t content_start = header_end + 4;

        size_t next = body.find("\r\n" + delimiter, content_start);
        if (next == std::string::npos) {
            return false;
        }

        const std::string content = body.substr(content_start, next - content_start);

        std::string name;
        std::string filename;
        size_t disp = raw_headers.find("name=\"");
        if (disp != std::string::npos) {
            size_t end = raw_headers.find('"', disp + 6);
            if (end != std::string::npos) {
                name = raw_headers.substr(disp + 6, end - disp - 6);
            }
        }
        size_t fn = raw_headers.find("filename=\"");
        if (fn != std::string::npos) {
            size_t end = raw_headers.find('"', fn + 10);
            if (end != std::string::npos) {
                filename = raw_headers.substr(fn + 10, end - fn - 10);
            }
        }

        if (!filename.empty()) {
            out.file_data = content;
            out.filename = filename;
        } else if (!name.empty()) {
            out.fields[name] = trim_ws(content);
        }

        pos = next + 2 + delimiter.size();
    }

    return true;
}

bool send_all(int fd, const std::string& data) {
    size_t sent = 0;
    while (sent < data.size()) {
        ssize_t n = ::send(fd, data.data() + sent, data.size() - sent, MSG_NOSIGNAL);
        if (n <= 0) {
            return false;
        }
        sent += static_cast<size_t>(n);
    }
    return true;
}

}  // namespace

void LogoUploadHTTPServer::runServer() {
    int listen_fd = ::socket(AF_INET, SOCK_STREAM, 0);
    if (listen_fd < 0) {
        std::cerr << "logo upload: socket() failed" << std::endl;
        running_ = false;
        return;
    }

    int reuse = 1;
    ::setsockopt(listen_fd, SOL_SOCKET, SO_REUSEADDR, &reuse, sizeof(reuse));

    struct sockaddr_in addr;
    std::memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(static_cast<uint16_t>(port_));
    // Loopback only: the service is a private backend behind the gateway.
    addr.sin_addr.s_addr = htonl(INADDR_LOOPBACK);

    if (::bind(listen_fd, reinterpret_cast<struct sockaddr*>(&addr), sizeof(addr)) < 0) {
        std::cerr << "logo upload: bind() failed on port " << port_ << std::endl;
        ::close(listen_fd);
        running_ = false;
        return;
    }

    if (::listen(listen_fd, 64) < 0) {
        std::cerr << "logo upload: listen() failed" << std::endl;
        ::close(listen_fd);
        running_ = false;
        return;
    }

    std::cout << "HTTP server listening on 127.0.0.1:" << port_ << std::endl;

    while (running_) {
        struct timeval timeout;
        timeout.tv_sec = 1;
        timeout.tv_usec = 0;
        ::setsockopt(listen_fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));

        int client_fd = ::accept(listen_fd, nullptr, nullptr);
        if (client_fd < 0) {
            continue;
        }

        std::string request;
        char buffer[8192];
        size_t header_end = std::string::npos;
        long long content_length = 0;

        while (true) {
            ssize_t n = ::recv(client_fd, buffer, sizeof(buffer), 0);
            if (n <= 0) {
                break;
            }
            request.append(buffer, static_cast<size_t>(n));

            if (header_end == std::string::npos) {
                header_end = request.find("\r\n\r\n");
                if (header_end != std::string::npos) {
                    std::string head = request.substr(0, header_end);
                    size_t cl = head.find("Content-Length:");
                    if (cl != std::string::npos) {
                        content_length = std::strtoll(head.c_str() + cl + 15, nullptr, 10);
                    }
                    if (content_length < 0 || content_length > 16LL * 1024 * 1024) {
                        break;
                    }
                }
            }

            if (header_end != std::string::npos &&
                request.size() >= header_end + 4 + static_cast<size_t>(content_length)) {
                break;
            }
        }

        if (header_end == std::string::npos) {
            ::close(client_fd);
            continue;
        }

        std::string head = request.substr(0, header_end);
        std::string body = request.substr(header_end + 4,
                                          static_cast<size_t>(content_length));

        size_t line_end = head.find("\r\n");
        std::string request_line = head.substr(0, line_end);
        size_t sp1 = request_line.find(' ');
        size_t sp2 = request_line.find(' ', sp1 + 1);
        std::string method = request_line.substr(0, sp1);
        std::string target = (sp2 == std::string::npos)
                                 ? request_line.substr(sp1 + 1)
                                 : request_line.substr(sp1 + 1, sp2 - sp1 - 1);

        std::map<std::string, std::string> headers;
        size_t hp = (line_end == std::string::npos) ? head.size() : line_end + 2;
        while (hp < head.size()) {
            size_t eol = head.find("\r\n", hp);
            if (eol == std::string::npos) break;
            std::string line = head.substr(hp, eol - hp);
            size_t colon = line.find(':');
            if (colon != std::string::npos) {
                std::string name = line.substr(0, colon);
                for (auto& ch : name) {
                    ch = static_cast<char>(std::tolower(static_cast<unsigned char>(ch)));
                }
                headers[name] = trim_ws(line.substr(colon + 1));
            }
            hp = eol + 2;
        }

        std::string path = target;
        std::string query;
        size_t q = target.find('?');
        if (q != std::string::npos) {
            path = target.substr(0, q);
            query = target.substr(q + 1);
        }

        std::string response_body;
        int status = 200;

        if (path == "/health" && (method == "GET" || method == "HEAD")) {
            response_body = createJSONResponse(true, "healthy");
        } else if (path == "/upload" && method == "POST") {
            std::string content_type = headers.count("content-type")
                                           ? headers["content-type"]
                                           : "";
            MultipartBody multipart;
            if (!parse_multipart(body, multipart_boundary(content_type), multipart)) {
                response_body = createJSONResponse(false, "malformed multipart body");
                status = 400;
            } else if (multipart.file_data.empty()) {
                response_body = createJSONResponse(false, "no file part in request");
                status = 400;
            } else {
                std::string token = multipart.fields.count("token_symbol")
                                        ? multipart.fields["token_symbol"]
                                        : "";
                std::string chain = multipart.fields.count("chain")
                                        ? multipart.fields["chain"]
                                        : "";

                std::vector<unsigned char> data(multipart.file_data.begin(),
                                                multipart.file_data.end());

                // The worker pool is asynchronous, so wait for the callback
                // rather than answering before the result is known.
                std::mutex done_mutex;
                std::condition_variable done_cv;
                bool done = false;
                LogoMetadata result;
                std::string error;

                service_->upload(token, chain, data, multipart.filename,
                    [&](const LogoMetadata& m, const std::string& err) {
                        std::lock_guard<std::mutex> lock(done_mutex);
                        result = m;
                        error = err;
                        done = true;
                        done_cv.notify_one();
                    });

                {
                    std::unique_lock<std::mutex> lock(done_mutex);
                    done_cv.wait_for(lock, std::chrono::seconds(30),
                                     [&] { return done; });
                }

                if (!done) {
                    response_body = createJSONResponse(false, "processing timeout");
                    status = 504;
                } else if (!error.empty()) {
                    response_body = createJSONResponse(false, error);
                    status = 400;
                } else {
                    response_body = "{\"success\":true,\"data\":" +
                                    metadata_to_json(result) + "}";
                }
            }
        } else if (path.rfind("/logo/", 0) == 0 && method == "GET") {
            std::string logo_id = path.substr(6);
            LogoMetadata m = service_->getMetadata(logo_id);
            if (m.id.empty()) {
                response_body = createJSONResponse(false, "not found");
                status = 404;
            } else {
                response_body = "{\"success\":true,\"data\":" + metadata_to_json(m) + "}";
            }
        } else if (path.rfind("/logo/", 0) == 0 && method == "DELETE") {
            std::string logo_id = path.substr(6);
            if (service_->deleteLogo(logo_id)) {
                response_body = createJSONResponse(true, "deleted");
            } else {
                response_body = createJSONResponse(false, "not found");
                status = 404;
            }
        } else if (path == "/logos" && method == "GET") {
            auto params = parseQueryParams(query);
            std::string token = params.count("token_symbol") ? params["token_symbol"] : "";
            auto logos = service_->getLogosByToken(token);

            std::string items = "[";
            for (size_t k = 0; k < logos.size(); ++k) {
                if (k) items += ",";
                items += metadata_to_json(logos[k]);
            }
            items += "]";
            response_body = "{\"success\":true,\"data\":" + items + "}";
        } else {
            response_body = createJSONResponse(false, "not found");
            status = 404;
        }

        const std::string status_text =
            (status == 200) ? "OK"
            : (status == 400) ? "Bad Request"
            : (status == 404) ? "Not Found"
            : (status == 504) ? "Gateway Timeout"
                              : "Error";

        std::string response =
            "HTTP/1.1 " + std::to_string(status) + " " + status_text + "\r\n"
            "Content-Type: application/json\r\n"
            "Content-Length: " + std::to_string(response_body.size()) + "\r\n"
            "Connection: close\r\n"
            "X-Content-Type-Options: nosniff\r\n"
            "\r\n" + response_body;

        send_all(client_fd, response);
        ::close(client_fd);
    }

    ::close(listen_fd);
}

void LogoUploadHTTPServer::handleRequest(
    const std::string& method,
    const std::string& path,
    const std::map<std::string, std::string>& headers,
    const std::string& body,
    std::string& response,
    int& status_code
) {
    if (path == "/health") {
        response = createJSONResponse(true, "healthy");
        status_code = 200;
        return;
    }
    
    if (path == "/upload" && method == "POST") {
        // Handle upload
        response = createJSONResponse(true, "Upload endpoint");
        status_code = 200;
        return;
    }
    
    response = createJSONResponse(false, "Not found");
    status_code = 404;
}

std::map<std::string, std::string> LogoUploadHTTPServer::parseQueryParams(const std::string& query) {
    std::map<std::string, std::string> params;
    
    size_t pos = 0;
    while (pos < query.size()) {
        size_t amp = query.find('&', pos);
        size_t eq = query.find('=', pos);
        
        if (eq == std::string::npos) break;
        
        std::string key = query.substr(pos, eq - pos);
        std::string value;
        
        if (amp != std::string::npos) {
            value = query.substr(eq + 1, amp - eq - 1);
            pos = amp + 1;
        } else {
            value = query.substr(eq + 1);
            pos = query.size();
        }
        
        params[key] = value;
    }
    
    return params;
}

std::string LogoUploadHTTPServer::createJSONResponse(bool success, const std::string& message, const std::string& data) {
    return "{\"success\":" + std::string(success ? "true" : "false") + 
           ",\"message\":\"" + message + "\"}";
}

// ============================================================================
// API Client Implementation
// ============================================================================

LogoServiceClient::LogoServiceClient(const std::string& base_url, const std::string& api_key)
    : base_url_(base_url), api_key_(api_key) {
    curl_ = curl_easy_init();
}

LogoServiceClient::~LogoServiceClient() {
    if (curl_) {
        curl_easy_cleanup(curl_);
    }
}

namespace {

/// Unescape a JSON string body (the characters json_escape writes).
std::string json_unescape(const std::string& v) {
    std::string out;
    out.reserve(v.size());
    for (size_t i = 0; i < v.size(); ++i) {
        if (v[i] != '\\' || i + 1 >= v.size()) {
            out.push_back(v[i]);
            continue;
        }
        char c = v[++i];
        switch (c) {
            case 'n': out.push_back('\n'); break;
            case 'r': out.push_back('\r'); break;
            case 't': out.push_back('\t'); break;
            case '"': out.push_back('"'); break;
            case '\\': out.push_back('\\'); break;
            case '/': out.push_back('/'); break;
            case 'u': {
                if (i + 4 >= v.size()) break;
                unsigned int cp = static_cast<unsigned int>(
                    std::strtoul(v.substr(i + 1, 4).c_str(), nullptr, 16));
                i += 4;
                if (cp < 0x80) {
                    out.push_back(static_cast<char>(cp));
                } else if (cp < 0x800) {
                    out.push_back(static_cast<char>(0xC0 | (cp >> 6)));
                    out.push_back(static_cast<char>(0x80 | (cp & 0x3F)));
                } else {
                    out.push_back(static_cast<char>(0xE0 | (cp >> 12)));
                    out.push_back(static_cast<char>(0x80 | ((cp >> 6) & 0x3F)));
                    out.push_back(static_cast<char>(0x80 | (cp & 0x3F)));
                }
                break;
            }
            default: out.push_back(c);
        }
    }
    return out;
}

}  // namespace

std::string LogoServiceClient::json_string_field(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return "";
    }
    pos += needle.size();
    while (pos < json.size() && (json[pos] == ' ' || json[pos] == '\t')) {
        ++pos;
    }
    if (pos >= json.size() || json[pos] != '"') {
        return "";
    }
    ++pos;

    std::string raw;
    bool escaped = false;
    for (; pos < json.size(); ++pos) {
        char c = json[pos];
        if (escaped) {
            raw.push_back('\\');
            raw.push_back(c);
            escaped = false;
            continue;
        }
        if (c == '\\') {
            escaped = true;
            continue;
        }
        if (c == '"') {
            break;
        }
        raw.push_back(c);
    }
    return json_unescape(raw);
}

int LogoServiceClient::json_int_field(const std::string& json, const std::string& key) {
    const std::string needle = "\"" + key + "\":";
    size_t pos = json.find(needle);
    if (pos == std::string::npos) {
        return 0;
    }
    return static_cast<int>(std::strtol(json.c_str() + pos + needle.size(), nullptr, 10));
}

/// Split a JSON array of flat objects into their individual object texts.
std::vector<std::string> LogoServiceClient::json_object_array(const std::string& json) {
    std::vector<std::string> objects;

    size_t start = json.find('[');
    if (start == std::string::npos) {
        return objects;
    }

    int depth = 0;
    bool in_string = false;
    bool escaped = false;
    size_t object_start = std::string::npos;

    for (size_t i = start + 1; i < json.size(); ++i) {
        char c = json[i];
        if (escaped) {
            escaped = false;
            continue;
        }
        if (c == '\\') {
            escaped = true;
            continue;
        }
        if (c == '"') {
            in_string = !in_string;
            continue;
        }
        if (in_string) {
            continue;
        }
        if (c == '{') {
            if (depth == 0) {
                object_start = i;
            }
            ++depth;
        } else if (c == '}') {
            --depth;
            if (depth == 0 && object_start != std::string::npos) {
                objects.push_back(json.substr(object_start, i - object_start + 1));
                object_start = std::string::npos;
            }
        } else if (c == ']' && depth == 0) {
            break;
        }
    }

    return objects;
}

LogoMetadata LogoServiceClient::parse_metadata(const std::string& json) {
    LogoMetadata m;
    m.id = json_string_field(json, "id");
    m.original_filename = json_string_field(json, "original_filename");
    m.stored_filename = json_string_field(json, "stored_filename");
    m.content_hash = json_string_field(json, "content_hash");
    m.mime_type = json_string_field(json, "mime_type");
    m.file_size = json_int_field(json, "file_size");
    m.width = json_int_field(json, "width");
    m.height = json_int_field(json, "height");
    m.thumbnail_url = json_string_field(json, "thumbnail_url");
    m.cdn_url = json_string_field(json, "cdn_url");
    m.created_at = json_string_field(json, "created_at");
    m.token_symbol = json_string_field(json, "token_symbol");
    m.chain = json_string_field(json, "chain");
    return m;
}

bool LogoServiceClient::performRequest(
    const std::string& method,
    const std::string& url,
    const std::string& body,
    const std::string& content_type,
    std::string& response
) {
    if (!curl_) {
        return false;
    }

    struct curl_slist* headers = nullptr;
    if (!api_key_.empty()) {
        headers = curl_slist_append(headers, ("X-API-Key: " + api_key_).c_str());
    }
    if (!content_type.empty()) {
        headers = curl_slist_append(headers, ("Content-Type: " + content_type).c_str());
    }

    response.clear();
    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers);
    curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION,
        [](void* contents, size_t size, size_t nmemb, void* userp) {
            ((std::string*)userp)->append((char*)contents, size * nmemb);
            return size * nmemb;
        });
    curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &response);
    curl_easy_setopt(curl_, CURLOPT_SSL_VERIFYPEER, 1L);
    curl_easy_setopt(curl_, CURLOPT_SSL_VERIFYHOST, 2L);

    if (method == "POST" || method == "PUT") {
        curl_easy_setopt(curl_, CURLOPT_CUSTOMREQUEST, method.c_str());
        curl_easy_setopt(curl_, CURLOPT_POSTFIELDSIZE, static_cast<long>(body.size()));
        curl_easy_setopt(curl_, CURLOPT_POSTFIELDS, body.data());
    } else if (method == "DELETE") {
        curl_easy_setopt(curl_, CURLOPT_CUSTOMREQUEST, "DELETE");
        curl_easy_setopt(curl_, CURLOPT_NOBODY, 0L);
    } else {
        curl_easy_setopt(curl_, CURLOPT_HTTPGET, 1L);
    }

    CURLcode res = curl_easy_perform(curl_);

    long status = 0;
    curl_easy_getinfo(curl_, CURLINFO_RESPONSE_CODE, &status);

    curl_slist_free_all(headers);

    return res == CURLE_OK && status >= 200 && status < 300;
}

bool LogoServiceClient::uploadLogo(
    const std::string& token_symbol,
    const std::string& chain,
    const std::string& file_path,
    LogoMetadata& metadata
) {
    std::string file_data = readFile(file_path);
    if (file_data.empty()) {
        return false;
    }

    // Derive a filename for the multipart part.
    std::string filename = file_path;
    size_t slash = filename.find_last_of('/');
    if (slash != std::string::npos) {
        filename = filename.substr(slash + 1);
    }

    const std::string boundary = "----TigerWalletBoundary" +
        std::to_string(static_cast<long long>(std::time(nullptr)));

    std::string body;
    auto add_field = [&body, &boundary](const std::string& name, const std::string& value) {
        body += "--" + boundary + "\r\n";
        body += "Content-Disposition: form-data; name=\"" + name + "\"\r\n\r\n";
        body += value + "\r\n";
    };
    add_field("token_symbol", token_symbol);
    add_field("chain", chain);

    body += "--" + boundary + "\r\n";
    body += "Content-Disposition: form-data; name=\"file\"; filename=\"" + filename + "\"\r\n";
    body += "Content-Type: application/octet-stream\r\n\r\n";
    body += file_data + "\r\n";
    body += "--" + boundary + "--\r\n";

    std::string response;
    if (!performRequest("POST", buildUploadURL(), body,
                        "multipart/form-data; boundary=" + boundary, response)) {
        return false;
    }

    metadata = parse_metadata(response);
    return !metadata.id.empty();
}

bool LogoServiceClient::getLogo(const std::string& logo_id, LogoMetadata& metadata) {
    std::string response;
    if (!performRequest("GET", buildRequestURL("/logo/" + logo_id), "",
                        "", response)) {
        return false;
    }

    metadata = parse_metadata(response);
    return !metadata.id.empty();
}

bool LogoServiceClient::deleteLogo(const std::string& logo_id) {
    std::string response;
    return performRequest("DELETE", buildRequestURL("/logo/" + logo_id), "",
                          "", response);
}

std::vector<LogoMetadata> LogoServiceClient::getLogos(const std::string& token_symbol) {
    std::vector<LogoMetadata> logos;

    std::string response;
    std::string url = buildRequestURL("/logos?token_symbol=" + token_symbol);
    if (!performRequest("GET", url, "", "", response)) {
        return logos;
    }

    for (const auto& object : json_object_array(response)) {
        logos.push_back(parse_metadata(object));
    }

    return logos;
}

std::string LogoServiceClient::buildUploadURL() {
    return base_url_ + "/upload";
}

std::string LogoServiceClient::buildRequestURL(const std::string& endpoint) {
    return base_url_ + endpoint;
}

std::string LogoServiceClient::readFile(const std::string& path) {
    std::ifstream file(path, std::ios::binary | std::ios::ate);
    
    if (!file.is_open()) {
        return "";
    }
    
    std::streamsize size = file.tellg();
    file.seekg(0, std::ios::beg);
    
    std::string buffer(size, '\0');
    if (!file.read(&buffer[0], size)) {
        return "";
    }
    
    return buffer;
}

// ============================================================================
// Main
// ============================================================================

int main() {
    // Initialize configuration
    LogoConfig config;
    config.storage_path = "/var/lib/tigerwallet/logos";
    config.cdn_base_url = "https://logos.tigerwallet.com";
    config.redis_host = "localhost";
    config.redis_port = 6379;
    config.max_file_size_mb = 10;
    config.allowed_extensions = {".png", ".jpg", ".jpeg", ".gif", ".webp"};
    config.allowed_mime_types = {"image/png", "image/jpeg", "image/gif", "image/webp"};
    config.thumbnail_sizes[0] = 16;
    config.thumbnail_sizes[1] = 32;
    config.thumbnail_sizes[2] = 64;
    config.thumbnail_sizes[3] = 128;
    config.worker_threads = 4;
    config.max_queue_size = 1000;
    config.enable_cdn = true;
    config.enable_cache = true;
    
    // Create and initialize service
    LogoUploadService service;
    if (!service.initialize(config)) {
        std::cerr << "Failed to initialize logo upload service" << std::endl;
        return 1;
    }
    
    // Start service
    service.start();
    
    // Start HTTP server
    LogoUploadHTTPServer http_server(&service, 8098);
    http_server.start();
    
    std::cout << "Logo upload service running. Press Ctrl+C to stop." << std::endl;
    
    // Wait for interrupt
    std::this_thread::sleep_for(std::chrono::hours(24));
    
    // Stop service
    http_server.stop();
    service.stop();
    
    return 0;
}
