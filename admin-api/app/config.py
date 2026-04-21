from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    database_url: str = "postgresql+asyncpg://llmrelay:llmrelay@localhost:5432/llmrelay"
    secret_key: str = "change-me-in-production"
    debug: bool = False

    class Config:
        env_file = "../.env"
        env_file_encoding = "utf-8"
        extra = "ignore"

settings = Settings()
