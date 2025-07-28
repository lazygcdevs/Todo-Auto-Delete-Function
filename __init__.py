import logging
import azure.functions as func
from azure.cosmos import CosmosClient, exceptions
from datetime import datetime, timedelta
import os
import json

def main(mytimer: func.TimerRequest) -> None:
    """
    Azure Function that runs every hour to delete documents from Cosmos DB
    that were created more than 10 hours ago.
    
    Args:
        mytimer: Timer trigger from Azure Functions
    """
    utc_timestamp = datetime.utcnow().replace(tzinfo=None).isoformat()
    
    if mytimer.past_due:
        logging.info('The timer is past due!')

    logging.info(f'Python timer trigger function ran at {utc_timestamp}')
    
    try:
        # Initialize Cosmos DB client
        cosmos_client = initialize_cosmos_client()
        
        # Get database and container references
        database = cosmos_client.get_database_client("todolist")
        
        # Get all containers in the database and clean each one
        containers = list(database.list_containers())
        total_deleted = 0
        
        for container_info in containers:
            container_name = container_info['id']
            logging.info(f"Processing container: {container_name}")
            
            container = database.get_container_client(container_name)
            deleted_count = delete_old_documents(container, container_name)
            total_deleted += deleted_count
            
        logging.info(f"Cleanup completed. Total documents deleted: {total_deleted}")
        
    except Exception as e:
        logging.error(f"Error during cleanup process: {str(e)}")
        raise


def initialize_cosmos_client():
    """
    Initialize and return a Cosmos DB client using connection string from environment variables.
    
    Returns:
        CosmosClient: Configured Cosmos DB client
        
    Raises:
        ValueError: If connection string is not found in environment variables
    """
    # Get connection string from environment variables
    connection_string = os.environ.get('COSMOS_DB_CONNECTION_STRING')
    
    if not connection_string:
        error_msg = "COSMOS_DB_CONNECTION_STRING environment variable not found"
        logging.error(error_msg)
        raise ValueError(error_msg)
    
    logging.info("Initializing Cosmos DB client...")
    return CosmosClient.from_connection_string(connection_string)


def delete_old_documents(container, container_name):
    """
    Delete documents from a specific container that were created more than 10 hours ago.
    
    Args:
        container: Cosmos DB container client
        container_name (str): Name of the container being processed
        
    Returns:
        int: Number of documents deleted
    """
    try:
        # Calculate the cutoff time (10 hours ago)
        cutoff_time = datetime.utcnow() - timedelta(hours=10)
        cutoff_timestamp = cutoff_time.isoformat() + "Z"
        
        logging.info(f"Deleting documents older than: {cutoff_timestamp}")
        
        # Query for old documents
        # This assumes documents have a '_ts' (timestamp) field or a custom 'createdAt' field
        # Cosmos DB automatically adds '_ts' field with Unix timestamp
        
        # Convert cutoff time to Unix timestamp for comparison with _ts
        cutoff_unix_timestamp = int(cutoff_time.timestamp())
        
        # Query documents older than 10 hours
        # Using _ts (system timestamp) which is automatically added by Cosmos DB
        query = f"SELECT c.id, c._ts FROM c WHERE c._ts < {cutoff_unix_timestamp}"
        
        logging.info(f"Executing query: {query}")
        
        # Get documents to delete
        documents_to_delete = list(container.query_items(
            query=query,
            enable_cross_partition_query=True
        ))
        
        deleted_count = 0
        
        if not documents_to_delete:
            logging.info(f"No old documents found in container '{container_name}'")
            return 0
            
        logging.info(f"Found {len(documents_to_delete)} documents to delete in container '{container_name}'")
        
        # Delete each document
        for doc in documents_to_delete:
            try:
                # Delete the document
                container.delete_item(
                    item=doc['id'],
                    partition_key=doc['id']  # Assuming id is the partition key, adjust if different
                )
                deleted_count += 1
                logging.debug(f"Deleted document with id: {doc['id']}")
                
            except exceptions.CosmosResourceNotFoundError:
                # Document was already deleted, skip
                logging.warning(f"Document {doc['id']} not found (may have been deleted already)")
                continue
                
            except Exception as e:
                logging.error(f"Failed to delete document {doc['id']}: {str(e)}")
                continue
        
        logging.info(f"Successfully deleted {deleted_count} documents from container '{container_name}'")
        return deleted_count
        
    except exceptions.CosmosResourceNotFoundError:
        logging.warning(f"Container '{container_name}' not found")
        return 0
        
    except Exception as e:
        logging.error(f"Error querying/deleting from container '{container_name}': {str(e)}")
        return 0


