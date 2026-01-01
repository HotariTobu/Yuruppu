# Backend Differences

## Gemini API

- Requires API key authentication
- Public access, simpler setup
- Model names: `gemini-2.5-flash`, etc.
- Environment: `GOOGLE_API_KEY`

## Vertex AI

- Requires GCP project and location
- Uses Application Default Credentials or service account
- Supports same model names plus third-party models
- Additional enterprise features (VPC-SC, CMEK)
- Environment: `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION`

The SDK automatically handles API differences based on the backend configuration.
