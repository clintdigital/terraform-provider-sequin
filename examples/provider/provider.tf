terraform {
  required_providers {
    sequin = {
      source  = "clintdigital/sequin"
      version = "~> 0.1"
    }
  }
}

provider "sequin" {
  endpoint = "https://your-instance.sequin.io" # or SEQUIN_ENDPOINT env var
  api_key  = var.sequin_api_key                # or SEQUIN_API_KEY env var
}
