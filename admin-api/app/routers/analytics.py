from decimal import Decimal
from fastapi import APIRouter, Depends
from sqlalchemy import select, func, Integer as sqlalchemy_integer
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.models import Request
from app.schemas import RequestOut, UsageStats

router = APIRouter(prefix="/analytics", tags=["analytics"])


@router.get("/requests", response_model=list[RequestOut])
async def list_requests(limit: int = 50, offset: int = 0, db: AsyncSession = Depends(get_db)):
    result = await db.execute(
        select(Request)
        .order_by(Request.created_at.desc())
        .limit(limit)
        .offset(offset)
    )
    return result.scalars().all()


@router.get("/stats", response_model=UsageStats)
async def usage_stats(db: AsyncSession = Depends(get_db)):
    # Aggregate stats in one query
    stats = await db.execute(
        select(
            func.count(Request.id).label("total_requests"),
            func.coalesce(func.sum(Request.prompt_tokens + Request.completion_tokens), 0).label("total_tokens"),
            func.coalesce(func.sum(Request.cost_usd), Decimal("0")).label("total_cost_usd"),
            func.coalesce(
                func.sum(func.cast(Request.cache_hit, sqlalchemy_integer)) * 1.0 / func.nullif(func.count(Request.id), 0),
                0
            ).label("cache_hit_rate"),
        )
    )
    row = stats.one()

    # Requests by model
    by_model_result = await db.execute(
        select(Request.model_id, func.count(Request.id))
        .where(Request.model_id.isnot(None))
        .group_by(Request.model_id)
    )
    by_model = {row[0]: row[1] for row in by_model_result}

    # Requests by provider
    by_provider_result = await db.execute(
        select(Request.provider, func.count(Request.id))
        .where(Request.provider.isnot(None))
        .group_by(Request.provider)
    )
    by_provider = {row[0]: row[1] for row in by_provider_result}

    return UsageStats(
        total_requests=row.total_requests,
        total_tokens=row.total_tokens,
        total_cost_usd=row.total_cost_usd,
        cache_hit_rate=float(row.cache_hit_rate or 0),
        requests_by_model=by_model,
        requests_by_provider=by_provider,
    )
