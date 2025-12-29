# Use an official Python runtime as a parent image
FROM python:3.9-slim

# Set the working directory in the container
WORKDIR /app

# Copy the requirements file into the container at /app
COPY ./requirements.txt /app/requirements.txt

# Install any needed packages specified in requirements.txt
# We add --no-cache-dir to reduce image size and --upgrade pip to ensure we have the latest version.
RUN pip install --no-cache-dir --upgrade pip -r /app/requirements.txt

# Copy the rest of the application's code into the container at /app
COPY . /app

# Make port 8000 available to the world outside this container
EXPOSE 8000

# Define environment variable to ensure python prints things without buffering
ENV PYTHONUNBUFFERED 1

# Run uvicorn server
# The host 0.0.0.0 makes the server accessible from outside the container
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
