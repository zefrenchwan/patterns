import requests
import json
import re
import urllib.parse
from elements import *
from graphs import *
from dto import *
from datetime import datetime, timedelta

base_url = 'http://localhost:8080'
# status 
status_url = base_url + "/status/"
# auth urls
tokens_url = base_url + "/token/"
user_upsert_url=base_url + "/user/upsert/"
# graphs management url
graph_create_url = base_url + "/graph/create/"
graph_clear_all_url = base_url + "/graph/all/clear/"
graph_add_import_url = base_url + "/graph/import/{0}/into/{1}/"
graphs_list_url = base_url + "/graph/list/"
graph_delete_url = base_url + "/graph/delete/{0}/"
graph_load_url = base_url + "/graph/load/{0}/"
graph_load_snapshot_url = base_url + "/graph/snapshot/{0}/at/{1}/"
graph_load_since_url = base_url + "/graph/slice/{0}/since/{1}/"
graph_load_between_url = base_url + "/graph/slice/{0}/between/{1}/and/{2}/"
element_upsert_url = base_url + "/elements/upsert/graph/{0}/"
element_load_url = base_url + "/elements/load/{0}/"
element_copy_url = base_url + "/elements/copy/{0}/to/{1}/"
element_delete_url = base_url + "/elements/delete/{0}/"
# find urls 
neighbors_url = base_url + "/find/neighbors/of/entities/for/trait/{0}/"
neighbors_url_since = base_url + "/find/neighbors/of/entities/for/trait/{0}/since/{1}"
neighbors_url_until = base_url + "/find/neighbors/of/entities/for/trait/{0}/until/{1}"
neighbors_url_between = base_url + "/find/neighbors/of/entities/for/trait/{0}/between/{1}/and/{2}"
# end of url

def datetime_to_api_string(value:datetime|str) -> str:
    """
    Apply datetime format to value, keep special values as is 
    """
    if value is None:
        return "];["
    if isinstance(value,str):
        return value
    return value.strftime("%Y-%m-%dT%H:%M:%S")


def check_api_connection() -> bool:
    """
    Test connection to patterns, and returns if app is responding
    """
    status = -1
    try:
        response = requests.get(status_url)
        status = response.status_code
    except:
        status = -1
    return status == 200


def generate_token(username:str, password:str) -> str|None:
    """
    Given username and password, generates a token from the api
    """
    response = requests.post(url=tokens_url, json = {"username": username, "password":password})
    if response.status_code != 200:
        print_response(response)
        return None
    values = json.loads(response.text)
    return values["token"]

    
def upsert_user(token: str, username: str, password: str) -> bool:
    """
    Upsert user. 
    Needs 'modifier' authorization to do so on existing user, or 'manager' to create a new one
    """
    response = requests.post(
        url=user_upsert_url, 
        headers= {"Authorization":"Bearer " + token},
        json = {"username": username, "password":password}
    )
    if response.status_code != 200:
        print_response(response)
        return False
    return True    


def list_graphs(token: str) -> list[Graph]|None:
    """
    List graphs. Loads metadata only
    """
    response = requests.get(graphs_list_url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return None
    result = list() 
    for element in response.json():
        value = authorized_graphs_from_json(element)
        result.append(value)
    return result 

    
def create_graph(token: str, name: str, description: str = "", metadata:dict[str,list[str]] = dict(), sources:list[str] = list()) -> str|None:
    """
    Creates a graph and returns its id
    """
    body = {"name":name}
    if len(description) != 0:
        body["description"] = description
    if len(metadata) != 0:
        body["metadata"] = metadata
    if len(sources) != 0:
        body["sources"] = sources

    response = requests.post(url=graph_create_url, headers= {"Authorization":"Bearer " + token}, json=body)
    if response.status_code != 200:
        print_response(response)
        return None
    return re.sub(r'\s+','', response.text).replace('"','')


def delete_graph(token: str, graph_id: str) -> bool:
    """
    Deletes graph by id, returns success (true of false)
    """
    url = graph_delete_url.format(graph_id)
    response = requests.delete(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return False
    return True


def load_graph_by_id(token: str, graph_id: str) -> Graph|None:
    """
    Load graph gets the full graph by id 
    """
    url = graph_load_url.format(graph_id)
    response = requests.get(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return None
    else:
        graph = graph_from_json(response.json())
        return graph 


def load_slice_graph_by_id(token: str, graph_id: str, start: datetime, end: datetime|None = None) -> Graph|None:
    url = None
    if end is not None:
        url =  graph_load_between_url.format(graph_id, datetime_to_api_string(start), datetime_to_api_string(end))
    else:
        url =  graph_load_since_url.format(graph_id, datetime_to_api_string(start))

    response = requests.get(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return None
    else:
        graph = graph_from_json(response.json())
        return graph 


def load_snapshot_graph_by_id(token: str, graph_id: str, moment: datetime) -> Graph|None:
    """
    Load graph gets the graph by id at a given moment. 
    It only loads active values. 
    """
    url = graph_load_snapshot_url.format(graph_id, datetime_to_api_string(moment))
    response = requests.get(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return None
    else:
        graph = graph_from_json(response.json())
        return graph 


def upsert_element_in_graph(token: str, graph_id: str, element: Element) -> bool:
    """
    Upsert element in a graph. 
    Because if may be a new element, graph id is mandatory to create the element
    """
    url = element_upsert_url.format(graph_id)
    response = requests.post(url=url, headers= {"Authorization":"Bearer " + token}, json=element.to_json())
    if response.status_code != 200:
        print_response(response)
        return False
    return True


def load_element_by_id(token: str, element_id: str) -> Element|None:
    """
    Load an element by id
    """
    url = element_load_url.format(element_id)
    response = requests.get(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return None
    else:
        element = element_from_json(response.json())
        return element


def copy_element_in_destination_graph(token: str, source_element_id: str, destination_graph_id: str) ->str|None:
    """
    Copy an element into another graph and returns the id of the new element
    """
    url = element_copy_url.format(source_element_id, destination_graph_id)
    response = requests.put(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return None 
    return re.sub(r'\s+','', response.text).replace('"','')
    

def delete_element(token: str, element_id: str) -> bool:
    """
    Deletes element by id, returns success (true of false)
    """
    url = element_delete_url.format(element_id)
    response = requests.delete(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return False
    return True


def add_imported_graph_to_current_graph(token:str, base_graph: str, imported_graph: str) -> bool:
    """
    Adds an imported graph to a graph
    """
    url = graph_add_import_url.format(imported_graph, base_graph)
    response = requests.put(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print_response(response)
        return False
    return True


def find_neighbors_of_requested_entities(token:str, trait: str, query = dict(), start = None, end = None) -> Graph:
    """
    Given a trait, and a map of attributes, find matching entities and get their neighbors
    """
    url = None 
    if start is not None and end is not None:
        start_value = datetime_to_api_string(start)
        end_value = datetime_to_api_string(end)
        url = neighbors_url_between.format(trait, start_value, end_value)
    elif start is not None:
        end_value = datetime_to_api_string(end)
        url = neighbors_url_until.format(trait, start_value, end_value)
    elif end is not None:
        start_value = datetime_to_api_string(start)
        url = neighbors_url_since.format(trait, start_value)
    else:
        url = neighbors_url.format(trait)    
    if query is not None and len(query) != 0:
        url = url + url + "?" + urllib.parse.urlencode(query)
    else:
        url = url + "/"
    # url is OK, then, call it 
    response = requests.get(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print("Error: " + response.text)
        return None
    else:
        graph = graph_from_json(response.json())
        return graph 
    

def clear_all_graphs(token:str) -> bool:
    """
    Clear any graph data
    """
    url = graph_clear_all_url
    response = requests.delete(url=url, headers= {"Authorization":"Bearer " + token})
    if response.status_code != 200:
        print("Error: " + response.text)
        return False
    return True


def print_response(response):
    status_code = response.status_code
    if status_code >= 400:
        print("ERROR! ", end = '')
    print("URL " + response.url + " returned " + str(status_code) +  "\n\t" + response.text +"\t" + str(response.headers))
    print()