from datetime import datetime
from decimal import Decimal
from typing import Optional
from pydantic import BaseModel


class ProjectCreate(BaseModel):
    name: str

class ProjectOut(BaseModel):
    id: int
    name: str
    created_at: datetime
    model_config = {"from_attributes": True}


class APIKeyCreate(BaseModel):
    name: str
    project_id: int
    rate_limit_rpm: int = 60
    budget_usd: Optional[Decimal] = None

class APIKeyOut(BaseModel):
    id: int
    name: str
    project_id: int
    rate_limit_rpm: int
    budget_usd: Optional[Decimal]
    spent_usd: Decimal
    is_active: bool
    created_at: datetime
    model_config = {"from_attributes": True}

class APIKeyWithSecret(APIKeyOut):
    key: str  # plaintext — returned only on creation, never again


class RequestOut(BaseModel):
    id: int
    provider: Optional[str]
    model_id: Optional[str]
    prompt_tokens: Optional[int]
    completion_tokens: Optional[int]
    cost_usd: Optional[Decimal]
    latency_ms: Optional[int]
    cache_hit: bool
    cache_type: Optional[str]
    status_code: Optional[int]
    created_at: datetime
    model_config = {"from_attributes": True}


class UsageStats(BaseModel):
    total_requests: int
    total_tokens: int
    total_cost_usd: Decimal
    cache_hit_rate: float
    requests_by_model: dict[str, int]
    requests_by_provider: dict[str, int]
