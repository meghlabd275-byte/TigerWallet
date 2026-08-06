# ECS Module Variables

variable "environment_name" {
  description = "Environment name"
  type        = string
}

variable "product_type" {
  description = "Product type"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
}

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
}

variable "backend_cpu" {
  description = "Backend container CPU units"
  type        = number
}

variable "backend_memory" {
  description = "Backend container memory in MB"
  type        = number
}

variable "backend_desired_count" {
  description = "Backend desired task count"
  type        = number
}

variable "frontend_cpu" {
  description = "Frontend container CPU units"
  type        = number
}

variable "frontend_memory" {
  description = "Frontend container memory in MB"
  type        = number
}

variable "frontend_desired_count" {
  description = "Frontend desired task count"
  type        = number
}

variable "alb_arn" {
  description = "ALB ARN"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs"
  type        = list(string)
}

variable "ecs_security_group_id" {
  description = "ECS security group ID"
  type        = string
}

variable "database_endpoint" {
  description = "Database endpoint"
  type        = string
}

variable "cache_endpoint" {
  description = "Cache endpoint"
  type        = string
}

variable "db_username" {
  description = "Database username"
  type        = string
}

variable "db_password" {
  description = "Database password"
  type        = string
}

variable "super_admin_url" {
  description = "Super Admin URL"
  type        = string
}

variable "api_key" {
  description = "API key"
  type        = string
}

variable "jwt_secret" {
  description = "JWT secret"
  type        = string
}

variable "backend_target_group_arn" {
  description = "Backend target group ARN"
  type        = string
  default     = ""
}

variable "frontend_target_group_arn" {
  description = "Frontend target group ARN"
  type        = string
  default     = ""
}

variable "tags" {
  description = "Tags"
  type        = map(string)
  default     = {}
}
