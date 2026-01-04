from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from sentence_transformers import SentenceTransformer, util
import torch
from typing import List, Dict, Any

# -----------------
# App Initialization
# -----------------

# Initialize the FastAPI app
app = FastAPI(
    title="Restaurant Language API",
    description="An API for various language processing tasks related to restaurant management.",
    version="1.1.0",
)

# Load the Sentence Transformer model
model = SentenceTransformer('all-MiniLM-L6-v2')

# -----------------
# Pydantic Models
# -----------------

class ProcessMessageRequest(BaseModel):
    task: str = Field(..., description="The task to perform.", example="check_duplicate")
    data: Dict[str, Any] = Field(..., description="The data required for the task.")

class ProcessMessageResponse(BaseModel):
    task: str = Field(..., description="The task that was performed.")
    result: Dict[str, Any] = Field(..., description="The result of the task.")

# -----------------
# API Endpoints
# -----------------

@app.get("/", summary="Health Check")
def read_root():
    """
    A simple health check endpoint to confirm the API is running.
    """
    return {"status": "ok"}


@app.post("/process-message", response_model=ProcessMessageResponse, summary="Process a natural language message")
def process_message(request: ProcessMessageRequest):
    """
    Processes a message by performing a specified task.

    Available tasks:
    - `check_duplicate`: Checks if a new restaurant name is a likely duplicate of one in an existing list.
      - **data required**: `new_name` (str), `existing_names` (List[str])
    """
    if request.task == "check_duplicate":
        # --- Duplicate Checking Logic ---
        new_name = request.data.get("new_name")
        existing_names = request.data.get("existing_names")

        if not new_name or not isinstance(existing_names, list):
            raise HTTPException(status_code=400, detail="Invalid data for check_duplicate task.")

        # Define the similarity threshold
        SIMILARITY_THRESHOLD = 0.8

        if not existing_names:
            return ProcessMessageResponse(task=request.task, result={"is_duplicate": False})

        # Encode the new name and the list of existing names into embeddings
        new_embedding = model.encode(new_name, convert_to_tensor=True)
        existing_embeddings = model.encode(existing_names, convert_to_tensor=True)

        # Compute cosine similarity
        cosine_scores = util.cos_sim(new_embedding, existing_embeddings)
        best_match_score, best_match_idx = torch.max(cosine_scores[0], dim=0)

        if best_match_score.item() > SIMILARITY_THRESHOLD:
            result = {
                "is_duplicate": True,
                "matched_name": existing_names[best_match_idx.item()],
                "similarity_score": round(best_match_score.item(), 4)
            }
        else:
            result = {"is_duplicate": False}
        
        return ProcessMessageResponse(task=request.task, result=result)

    else:
        raise HTTPException(status_code=400, detail=f"Unknown task: {request.task}")
