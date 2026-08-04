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
        // Simplified JPEG dimension reading
        width = 0;
        height = 0;
        return true;
    }
    
    if (format == "gif") {
        mime_type = "image/gif";
        if (data.size() >= 10) {
            width = data[6] | (data[7] << 8);
            height = data[8] | (data[9] << 8);
        }
        return true;
    }
    
    if (format == "webp") {
        mime_type = "image/webp";
        return true;
    }
    
    return false;
}

std::vector<unsigned char> ImageProcessor::createThumbnail(
    const std::vector<unsigned char>& image_data,
    int target_width,
    int target_height
) {
    // In production, this would use a library like stb_image or libvips
    // For now, return the original data as placeholder
    return image_data;
}

std::vector<unsigned char> ImageProcessor::convertToPNG(const std::vector<unsigned char>& data) {
    // In production, use image conversion library
    return data;
}

std::vector<unsigned char> ImageProcessor::optimizePNG(const std::vector<unsigned char>& data) {
    // In production, use pngquant or similar
    return data;
}

bool ImageProcessor::resizeImage(
    const std::vector<unsigned char>& input,
    std::vector<unsigned char>& output,
    int target_width,
    int target_height
) {
    // In production, use proper image resizing library
    output = input;
    return true;
}

// ============================================================================
// Local Storage Backend Implementation
// ============================================================================

bool LocalStorageBackend::initialize(const LogoConfig& config) {
    config_ = config;
    
    // Create storage directory if it doesn't exist
    std::string cmd = "mkdir -p " + config_.storage_path;
    system(cmd.c_str());
    
    return true;
}

bool LocalStorageBackend::store(const std::string& path, const std::vector<unsigned char>& data) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::string full_path = config_.storage_path + "/" + path;
    
    // Create directory structure
    size_t last_slash = full_path.find_last_of('/');
    if (last_slash != std::string::npos) {
        std::string dir = full_path.substr(0, last_slash);
        std::string cmd = "mkdir -p " + dir;
        system(cmd.c_str());
    }
    
    std::ofstream file(full_path, std::ios::binary);
    if (!file.is_open()) {
        return false;
    }
    
    file.write(reinterpret_cast<const char*>(data.data()), data.size());
    file.close();
    
    return true;
}

std::vector<unsigned char> LocalStorageBackend::retrieve(const std::string& path) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::string full_path = config_.storage_path + "/" + path;
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
    
    std::string full_path = config_.storage_path + "/" + path;
    return remove(full_path.c_str()) == 0;
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

bool S3StorageBackend::initialize(const LogoConfig& config) {
    config_ = config;
    bucket_ = config.aws_bucket;
    region_ = config.aws_region;
    return curl_ != nullptr;
}

bool S3StorageBackend::store(const std::string& path, const std::vector<unsigned char>& data) {
    std::lock_guard<std::mutex> lock(curl_mutex_);
    
    // In production, implement proper S3 multipart upload
    return uploadToS3(path, data);
}

bool S3StorageBackend::uploadToS3(const std::string& path, const std::vector<unsigned char>& data) {
    if (!curl_) return false;
    
    std::string url = "https://" + bucket_ + ".s3." + region_ + ".amazonaws.com/" + path;
    
    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl_, CURLOPT_POST, 1L);
    curl_easy_setopt(curl_, CURLOPT_POSTFIELDSIZE, data.size());
    curl_easy_setopt(curl_, CURLOPT_POSTFIELDS, data.data());
    
    struct curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, "Content-Type: image/png");
    
    curl_easy_setopt(curl_, CURLOPT_HTTPHEADER, headers);
    
    CURLcode res = curl_easy_perform(curl_);
    
    curl_slist_free_all(headers);
    
    return res == CURLE_OK;
}

std::vector<unsigned char> S3StorageBackend::retrieve(const std::string& path) {
    std::lock_guard<std::mutex> lock(curl_mutex_);
    
    if (!curl_) return {};
    
    std::string url = "https://" + bucket_ + ".s3." + region_ + ".amazonaws.com/" + path;
    
    curl_easy_setopt(curl_, CURLOPT_URL, url.c_str());
    
    std::string response;
    curl_easy_setopt(curl_, CURLOPT_WRITEFUNCTION, 
        [](void* contents, size_t size, size_t nmemb, void* userp) {
            ((std::string*)userp)->append((char*)contents, size * nmemb);
            return size * nmemb;
        });
    curl_easy_setopt(curl_, CURLOPT_WRITEDATA, &response);
    
    CURLcode res = curl_easy_perform(curl_);
    
    if (res != CURLE_OK) {
        return {};
    }
    
    return std::vector<unsigned char>(response.begin(), response.end());
}

bool S3StorageBackend::remove(const std::string& path) {
    // Implement S3 delete
    return true;
}

std::string S3StorageBackend::getURL(const std::string& path) {
    return "https://" + bucket_ + ".s3." + region_ + ".amazonaws.com/" + path;
}

std::string S3StorageBackend::generatePresignedURL(const std::string& path) {
    // In production, generate proper presigned URL with AWS Signature
    return getURL(path);
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

RedisCache::RedisCache() : redis_context_(nullptr), connected_(false) {}

RedisCache::~RedisCache() {
    // Close connection
}

bool RedisCache::connect(const std::string& host, int port) {
    // In production, use hiredis or redis-plus-plus
    config_.redis_host = host;
    config_.redis_port = port;
    connected_ = true;
    return true;
}

bool RedisCache::set(const std::string& key, const std::string& value, int ttl_seconds) {
    if (!connected_) return false;
    // In production, implement Redis SET
    return true;
}

std::string RedisCache::get(const std::string& key) {
    if (!connected_) return "";
    // In production, implement Redis GET
    return "";
}

bool RedisCache::del(const std::string& key) {
    if (!connected_) return false;
    return true;
}

bool RedisCache::exists(const std::string& key) {
    if (!connected_) return false;
    return false;
}

bool RedisCache::ping() {
    return connected_;
}

std::string RedisCache::buildRedisCommand(const std::vector<std::string>& args) {
    // In production, implement proper RESP protocol
    return "";
}

bool RedisCache::sendCommand(const std::string& command) {
    return true;
}

std::string RedisCache::readResponse() {
    return "";
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
            } else {
                total_failed_++;
            }
        } catch (const std::exception& e) {
            std::cerr << "Error processing task: " << e.what() << std::endl;
            total_failed_++;
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

void LogoUploadHTTPServer::runServer() {
    // In production, use proper HTTP server library
    std::cout << "HTTP server running on port " << port_ << std::endl;
    
    while (running_) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }
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
    
    std::vector<unsigned char> data(file_data.begin(), file_data.end());
    
    // In production, make actual HTTP request to upload service
    return true;
}

bool LogoServiceClient::getLogo(const std::string& logo_id, LogoMetadata& metadata) {
    // In production, make actual HTTP request
    return false;
}

bool LogoServiceClient::deleteLogo(const std::string& logo_id) {
    // In production, make actual HTTP request
    return false;
}

std::vector<LogoMetadata> LogoServiceClient::getLogos(const std::string& token_symbol) {
    return {};
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
