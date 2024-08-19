from api import *
from elements import *
from uuid import uuid4
from connection_data import *

def test_default_user_connection():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"