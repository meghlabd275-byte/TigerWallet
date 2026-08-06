# Terraform Variables

variable "aws_region" {
  description = "AWS region"
  type        = string
  default    = "us-east-1"
}

variable "environment_name" {
  description = "Environment name"
  type        = string
}

variable "product_type" {
  description = "Product type: master_wallet, user_wallet, bots, project_party"
  type        = string
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "List of availability zones"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

# Database
variable "db_name" {
  description = "Database name"
  type        = string
}

variable "db_username" {
  description = "Database username"
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}

variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.medium"
}

variable "db_allocated_storage" {
  description = "Allocated storage in GB"
  type        = number
  default     = 50
}

# Redis
variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.medium"
}

variable "redis_num_nodes" {
  description = "Number of Redis nodes"
  type        = number
  default     = 2
}

# Container
variable "ecr_repository_url" {
  description = "ECR repository URL"
  type        = string
}

variable "container_image" {
  description = "Container image"
  type        = string
}

variable "container_port" {
  description = "Container port"
  type        = number
  default     = 8000
}

# Backend scaling
variable "backend_cpu" {
  description = "Backend container CPU units"
  type        = number
  default     = 512
}

variable "backend_memory" {
  description = "Backend container memory in MB"
  type        = number
  default     = 1024
}

variable "backend_desired_count" {
  description = "Backend desired task count"
  type        = number
  default     = 2
}

# Frontend scaling
variable "frontend_cpu" {
  description = "Frontend container CPU units"
  type        = number
  default     = 256
}

variable "frontend_memory" {
  description = "Frontend container memory in MB"
  type        = number
  default     = 512
}

variable "frontend_desired_count" {
  description = "Frontend desired task count"
  type        = number
  default     = 2
}

# Authentication
variable "super_admin_url" {
  description = "Super Admin URL"
  type        = string
}

variable "api_key" {
  description = "API key for Super Admin"
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "JWT secret"
  type        = string
  sensitive   = true
}

# Tags
variable "common_tags" {
  description = "Common tags"
  type        = map(string)
  default     = {}
}
