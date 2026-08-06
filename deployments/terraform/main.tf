# TigerWallet White Label Product Terraform Module
# Deploy any white label product on AWS

terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  
  backend "s3" {
    bucket = "tigerwallet-terraform-state"
    key    = "white-label/deployment.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = var.aws_region
}

# ============== VPC Module ==============
module "vpc" {
  source = "./modules/vpc"
  
  environment_name = var.environment_name
  vpc_cidr         = var.vpc_cidr
  
  availability_zones = var.availability_zones
  
  tags = var.common_tags
}

# ============== RDS Module ==============
module "database" {
  source = "./modules/rds"
  
  environment_name = var.environment_name
  db_name          = var.db_name
  db_username      = var.db_username
  db_password      = var.db_password
  db_instance_class = var.db_instance_class
  db_allocated_storage = var.db_allocated_storage
  
  vpc_id           = module.vpc.vpc_id
  subnet_ids       = module.vpc.private_subnet_ids
  security_group_id = module.vpc.database_security_group_id
  
  tags = var.common_tags
}

# ============== ElastiCache Module ==============
module "cache" {
  source = "./modules/elasticache"
  
  environment_name = var.environment_name
  node_type        = var.redis_node_type
  num_cache_nodes = var.redis_num_nodes
  
  vpc_id           = module.vpc.vpc_id
  subnet_ids       = module.vpc.private_subnet_ids
  security_group_id = module.vpc.redis_security_group_id
  
  tags = var.common_tags
}

# ============== ALB Module ==============
module "alb" {
  source = "./modules/alb"
  
  environment_name = var.environment_name
  
  vpc_id           = module.vpc.vpc_id
  public_subnet_ids = module.vpc.public_subnet_ids
  security_group_id = module.vpc.alb_security_group_id
  
  tags = var.common_tags
}

# ============== ECS Module ==============
module "ecs" {
  source = "./modules/ecs"
  
  environment_name = var.environment_name
  product_type    = var.product_type  # master_wallet, user_wallet, bots, project_party
  
  ecr_repository_url = var.ecr_repository_url
  
  container_image = var.container_image
  container_port  = var.container_port
  
  backend_cpu    = var.backend_cpu
  backend_memory = var.backend_memory
  backend_desired_count = var.backend_desired_count
  
  frontend_cpu    = var.frontend_cpu
  frontend_memory = var.frontend_memory
  frontend_desired_count = var.frontend_desired_count
  
  alb_arn = module.alb.arn
  
  vpc_id           = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  ecs_security_group_id = module.vpc.ecs_security_group_id
  
  database_endpoint = module.database.endpoint
  cache_endpoint    = module.cache.endpoint
  
  super_admin_url = var.super_admin_url
  api_key         = var.api_key
  jwt_secret      = var.jwt_secret
  
  tags = var.common_tags
}

# ============== Outputs ==============
output "alb_dns_name" {
  description = "ALB DNS Name"
  value       = module.alb.dns_name
}

output "database_endpoint" {
  description = "Database endpoint"
  value       = module.database.endpoint
}

output "cache_endpoint" {
  description = "Cache endpoint"
  value       = module.cache.endpoint
}

output "ecs_cluster_name" {
  description = "ECS Cluster Name"
  value       = module.ecs.cluster_name
}
