from api import *
from elements import *
from uuid import uuid4
from connection_data import *


def test_graph_metadata_use():
    """
    Validates that metadata is saved, and loaded back
    """
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    metadata = {"author":["me"], "date":["now"]}

    graph_id = create_graph(token,"first graph", "description test", metadata)
    assert graph_id is not None, "failed to create graph"
    result = load_graph_by_id(token, graph_id)
    assert result.graph_id == graph_id
    assert result.name == "first graph"
    assert result.description == "description test"
    assert result.metadata == metadata

    loaded_graphs = list_graphs(token)
    assert len(loaded_graphs) == 1
    loaded_graph = loaded_graphs[0]
    assert loaded_graph.graph_id == graph_id
    assert loaded_graph.name == "first graph"
    assert loaded_graph.description == "description test"
    assert loaded_graph.metadata == metadata

    # clean after test
    assert clear_all_graphs(token)

def test_graph_deletion():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    graph_id = create_graph(token,"first graph")
    assert graph_id is not None, "failed to create graph"
    child_graph_id = create_graph(token,"child graph", sources=[graph_id])    
    assert child_graph_id is not None, "failed to create graph"

    # insert two elements in source graph, one in child
    # It means that deleting source should fail due to dependency. 
    france = Element(id = str(uuid4()))
    france.traits = ["Country"]
    france.activity = ["]-oo;+oo["]
    france.add_attribute_value("name", "France")
    upsert_element_in_graph(token, graph_id, france)
 
    paris = Element(id = str(uuid4()))
    paris.traits = ["City"]
    paris.activity = ["]-oo;+oo["]
    paris.add_attribute_value("name", "Paris")
    upsert_element_in_graph(token, graph_id, paris)
 
    capitale = Element(id = str(uuid4()))
    capitale.traits = ["Capitale"]
    capitale.activity = ["]-oo;+oo["]
    capitale.add_role_value("subject", paris.element_id)
    capitale.add_role_value("object", france.element_id)
    upsert_element_in_graph(token, child_graph_id, capitale)

    # check insertions went well 
    source_graph = load_graph_by_id(token, graph_id)
    child_graph = load_graph_by_id(token, child_graph_id)
    assert len(source_graph.elements) == 2
    assert len(child_graph.elements) == 3
    
    # cannot delete because of dependency
    assert not delete_graph(token, graph_id)
    assert delete_graph(token, child_graph_id)
    assert delete_graph(token, graph_id)

    # and of course, clean
    assert clear_all_graphs(token)


def test_graph_multi_layer():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    graph_id = create_graph(token,"first graph")
    assert graph_id is not None, "cannot create graph"

    # insert elements in layer graph
    france = Element(id = str(uuid4()))
    france.traits = ["Country"]
    france.activity = ["]-oo;+oo["]
    france.add_attribute_value("name", "France")

    paris = Element(id = str(uuid4()))
    paris.traits = ["City"]
    paris.activity = ["]-oo;+oo["]
    paris.add_attribute_value("name", "Paris")

    capitale = Element(id = str(uuid4()))
    capitale.traits = ["Capitale"]
    capitale.activity = ["]-oo;+oo["]
    capitale.add_role_value("subject", paris.element_id)
    capitale.add_role_value("object", france.element_id)

    exception_upsert = False
    for i in range (0,3):
        try:
            upsert_element_in_graph(token, graph_id, france)
            upsert_element_in_graph(token, graph_id, paris)
            upsert_element_in_graph(token, graph_id, capitale)
        except:
            exception_upsert = True
    assert not exception_upsert, "failed to upsert data in source graph"

    # test individual load, entity, relation and non existing
    element = load_element_by_id(token, paris.element_id)
    assert element is not None, "expecting one element from loading element"
    assert element == paris
    element = load_element_by_id(token, capitale.element_id)
    assert element is not None, "expecting one element from loading element"
    assert element == capitale
    element = load_element_by_id(token, "i don't exist")
    assert element is None, "loading element finds non existing element"
    

    # test graph load for a single graph
    no_graph = load_element_by_id(token, "i don't exist either")
    assert no_graph is None, "no graph in database, but a graph is loaded"
    source_loaded = load_graph_by_id(token, graph_id)
    assert source_loaded is not None, "failing to load graph" 

    expected_graph = Graph(graph_id, "first graph")
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(capitale, graph_id, True)
    expected_graph.add_node(paris, graph_id, True)
    assert expected_graph == source_loaded, "comparison failure between expected and source graph"


    # Adding layer over source graph
    other_graph_id = create_graph(token = token, name = "child graph", sources=[graph_id])
    assert other_graph_id is not None, "failing to load child graph"

    important_event = Element(id = str(uuid4()))
    important_event.traits = ["Event"]
    important_event.activity = ["]-oo;+oo["]
    important_event.add_attribute_value("name", "A long event")

    capitale_link = Element(id = str(uuid4()))
    capitale_link.traits = ["Host"]
    capitale_link.activity = ["]-oo;+oo["]
    capitale_link.add_role_value("subject", paris.element_id)
    capitale_link.add_role_value("object", important_event.element_id)

    exception_upsert = False 
    for i in range(0,3):
        try:
            upsert_element_in_graph(token, other_graph_id, important_event)
            upsert_element_in_graph(token, other_graph_id, capitale_link)
        except:
            exception_upsert = True 
    assert not exception_upsert, "failed to load child graph"

    oopsie = Element(id = str(uuid4()))
    oopsie.traits = ['Oops']
    oopsie.activity = ["]-oo;+oo["]
    upsert_element_in_graph(token, other_graph_id, oopsie)
    assert delete_element(token, oopsie.element_id), "failed to delete element"
    

    top_layer_loaded = load_graph_by_id(token, other_graph_id)
    assert top_layer_loaded is not None, "failed to load child graph"
    assert len(top_layer_loaded) == 5, "missing elements in child graph"

    ########################################
    ## TEST 2: compare graphs with layers ##
    ########################################
    expected_graph = Graph(other_graph_id, name = "child graph")
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(capitale, graph_id, True)
    expected_graph.add_node(paris, graph_id, True)
    expected_graph.add_node(important_event, other_graph_id, True)
    expected_graph.add_node(capitale_link, other_graph_id, True)
    assert expected_graph == top_layer_loaded

    # test graphs listing
    all_graphs = list_graphs(token)
    assert len(all_graphs) == 2, "missing graphs from loaded values, got " + str(len(all_graphs))
    assert {source_loaded.graph_id, top_layer_loaded.graph_id} == {graph.graph_id for graph in all_graphs}

    ## delete graph is possible    
    assert delete_graph(token, other_graph_id), "failed to delete child graph"
    all_graphs = list_graphs(token)
    assert len(all_graphs) == 1, "missing graphs from loaded values, got " + str(len(all_graphs))
    assert {source_loaded.graph_id} == {graph.graph_id for graph in all_graphs}

    # remaining data, so just clean for next tests
    assert clear_all_graphs(token)