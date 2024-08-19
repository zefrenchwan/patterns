import sys
from test_elements_crud import *
from test_graph_crud import *
from test_connection import *
from test_graph_snapshot import *
from test_graph_slices import *
from test_graph_copy import *
from test_neighbors import *


def usage(args: list[str]):
    """
    Test parameters and quit if invalid
    """
    if len(args) != 2:
        print("usage: username userpass")
        sys.exit(-1)

if __name__ == '__main__':
    args = sys.argv[1:]
    usage(args)
    username = args[0]
    userpass = args[1]

    # generate token
    assert check_api_connection(), "no connection to patterns"
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"
    
    test_dynamic_import_graph()
