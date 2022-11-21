terraform {
  required_version = ">= 0.12.29"
  required_providers {
    ec = {
      source = "elastic/ec"
      version = "0.2.1"
    }
  }
}

provider "ec" {
  apikey = var.ec_api_key
}

variable "ec_api_key" {
  type = string
}

variable "deployment_name" {
  type = string
}

variable "deployment_template_id" {
  type = string
}

variable "cloud_region" {
  type = string
}

data "ec_stack" "latest" {
  version_regex = "latest"
  region = var.cloud_region
}

resource "ec_deployment" "elasticsearch" {
  name = var.deployment_name
  deployment_template_id = var.deployment_template_id
  region = data.ec_stack.latest.region
  version = data.ec_stack.latest.version
  elasticsearch {
    autoscale = "true"
    topology {
      id = "hot_content"
      size = "8g"
      zone_count = "2"
    }
  }
  kibana {
    topology {
      size = "4g"
      zone_count = "2"
    }
  }
}

output "elasticsearch_endpoint" {
  value = ec_deployment.elasticsearch.elasticsearch[0].https_endpoint
}

output "kibana_endpoint" {
  value = ec_deployment.elasticsearch.kibana[0].https_endpoint
}

output "elasticsearch_username" {
  value = ec_deployment.elasticsearch.elasticsearch_username
}

output "elasticsearch_password" {
  value = ec_deployment.elasticsearch.elasticsearch_password
  sensitive = true
}
