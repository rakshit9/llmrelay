import hashlib
import secrets
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_db
from app.models import APIKey
from app.schemas import APIKeyCreate, APIKeyOut, APIKeyWithSecret

router = APIRouter(prefix="/keys", tags=["api-keys"])


def _hash_key(key: str) -> str:
    return hashlib.sha256(key.encode()).hexdigest()


@router.get("/", response_model=list[APIKeyOut])
async def list_keys(project_id: int | None = None, db: AsyncSession = Depends(get_db)):
    q = select(APIKey).order_by(APIKey.created_at.desc())
    if project_id:
        q = q.where(APIKey.project_id == project_id)
    result = await db.execute(q)
    return result.scalars().all()


@router.post("/", response_model=APIKeyWithSecret, status_code=201)
async def create_key(body: APIKeyCreate, db: AsyncSession = Depends(get_db)):
    # Generate a secure random key — shown once, never stored plaintext
    raw_key = "sk-relay-" + secrets.token_urlsafe(32)
    key = APIKey(
        project_id=body.project_id,
        name=body.name,
        key_hash=_hash_key(raw_key),
        rate_limit_rpm=body.rate_limit_rpm,
        budget_usd=body.budget_usd,
    )
    db.add(key)
    await db.commit()
    await db.refresh(key)
    return APIKeyWithSecret(**APIKeyOut.model_validate(key).model_dump(), key=raw_key)


@router.patch("/{key_id}/revoke", response_model=APIKeyOut)
async def revoke_key(key_id: int, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(APIKey).where(APIKey.id == key_id))
    key = result.scalar_one_or_none()
    if not key:
        raise HTTPException(status_code=404, detail="Key not found")
    key.is_active = False
    await db.commit()
    await db.refresh(key)
    return key


@router.delete("/{key_id}", status_code=204)
async def delete_key(key_id: int, db: AsyncSession = Depends(get_db)):
    result = await db.execute(select(APIKey).where(APIKey.id == key_id))
    key = result.scalar_one_or_none()
    if not key:
        raise HTTPException(status_code=404, detail="Key not found")
    await db.delete(key)
    await db.commit()
