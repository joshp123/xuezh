terraform {
  required_version = ">= 1.6.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
    azapi = {
      source  = "Azure/azapi"
      version = "~> 1.12"
    }
  }
}

provider "azurerm" {
  features {}
}

provider "azapi" {}

variable "subscription_id" {
  type        = string
  description = "Azure subscription ID"
}

variable "resource_group_name" {
  type        = string
  default     = "rg-xuezh"
  description = "Resource group for speech services"
}

variable "location" {
  type        = string
  default     = "westeurope"
  description = "Azure region closest to NL (use westeurope unless policy blocks)"
}

variable "speech_name" {
  type        = string
  description = "Globally unique Speech resource name"
}

resource "azurerm_resource_group" "xuezh" {
  name     = var.resource_group_name
  location = var.location
}

resource "azapi_resource" "speech" {
  type      = "Microsoft.CognitiveServices/accounts@2023-05-01"
  name      = var.speech_name
  location  = azurerm_resource_group.xuezh.location
  parent_id = azurerm_resource_group.xuezh.id

  body = jsonencode({
    kind = "SpeechServices"
    sku  = { name = "F0" }
    properties = {}
  })
}

output "speech_resource_id" {
  value = azapi_resource.speech.id
}
