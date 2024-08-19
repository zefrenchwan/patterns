from api import *
from elements import *
from uuid import uuid4
from connection_data import *


def test_graph_copy_entity():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    graph_id = create_graph(token,"first graph")
    assert graph_id is not None, "failed to create graph"
    
    france = Element(id = str(uuid4()))
    france.traits = ["Country"]
    france.activity = ["]-oo;+oo["]
    france.add_attribute_value("name", "France")
    upsert_element_in_graph(token, graph_id, france)

    other_graph_id = create_graph(token,"child graph", sources=[graph_id])
    assert other_graph_id is not None, "failed to create graph"
    
    new_id = copy_element_in_destination_graph(token, france.element_id, other_graph_id)
    assert new_id is not None, "cannot create copy"

    # load new element and test it
    new_element = load_element_by_id(token, new_id)
    new_element_id = new_element.element_id 
    assert new_element is not None, "cannot find element back"
    # except id, should be the same 
    new_element.element_id = france.element_id
    assert new_element == france 
    # test is done, set the id back
    new_element.element_id = new_element_id

    # element inserted, check its data
    new_graph = load_graph_by_id(token, other_graph_id)
    assert new_graph is not None, "cannot load child graph"

    expected_graph = Graph(other_graph_id, name = "child graph")
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(new_element, other_graph_id, True, france.element_id, graph_id)
    assert expected_graph == new_graph

    assert clear_all_graphs(token)



def test_graph_copy_relation():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    graph_id = create_graph(token,"first graph")
    assert graph_id is not None, "failed to create graph"
    
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
    upsert_element_in_graph(token, graph_id, capitale)

    other_graph_id = create_graph(token,"child graph", sources=[graph_id])
    assert other_graph_id is not None, "failed to create graph"
    
    new_id = copy_element_in_destination_graph(token, capitale.element_id, other_graph_id)
    assert new_id is not None, "cannot create copy"

    # load new element and test it
    new_element = load_element_by_id(token, new_id)
    new_element_id = new_element.element_id 
    assert new_element is not None, "cannot find element back"
    # except id, should be the same 
    new_element.element_id = capitale.element_id
    assert new_element == capitale 
    # test is done, set the id back
    new_element.element_id = new_element_id

    # element inserted, check its data
    new_graph = load_graph_by_id(token, other_graph_id)
    assert new_graph is not None, "cannot load child graph"

    expected_graph = Graph(other_graph_id, name = "child graph")
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(paris, graph_id, True)
    expected_graph.add_node(capitale, graph_id, True)
    expected_graph.add_node(new_element, other_graph_id, True, capitale.element_id, graph_id)
    assert expected_graph == new_graph

    assert clear_all_graphs(token)
