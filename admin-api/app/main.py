from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.routers import projects, api_keys, analytics

app = FastAPI(title="LLM Relay Admin API", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000", "http://localhost:3001"],
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(projects.router)
app.include_router(api_keys.router)
app.include_router(analytics.router)

@app.get("/health")
async def health():
    return {"status": "ok"}
