# Azure Speech resource (Terraform)

Creates an Azure Speech (Cognitive Services) resource in the region closest to NL.

## Prereqs
- Azure account + subscription
- Azure CLI (`az`) logged in
  - OpenTofu (Terraform-compatible)

All tools are provided via `devenv`.

## Usage

```bash
cd infra/azure/speech

# login + select subscription
az login
az account set --subscription <SUBSCRIPTION_ID>

tofu init
tofu apply \
  -var "subscription_id=<SUBSCRIPTION_ID>" \
  -var "speech_name=xuezh-speech-weu-<unique>"
```

## Notes
- Free tier still requires auth keys; grab `AZURE_SPEECH_KEY` and `AZURE_SPEECH_REGION` from the Azure portal
  under the Speech resource -> "Keys and Endpoint".
- If `westeurope` is blocked, try `northeurope`.
