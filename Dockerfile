FROM python:3.13.3-slim-bookworm@sha256:56a11364ffe0fee3bd60af6d6d5209eba8a99c2c16dc4c7c5861dc06261503cc
RUN apt-get update && apt-get install -y --no-install-recommends poppler-utils util-linux ca-certificates && rm -rf /var/lib/apt/lists/*
ARG HUB_UID=1000
ARG HUB_GID=1000
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 HOME=/tmp
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY paperwork ./paperwork
USER ${HUB_UID}:${HUB_GID}
CMD ["python", "-m", "uvicorn", "paperwork.server:app", "--host", "127.0.0.1", "--port", "8088", "--no-access-log"]
