from api import *
from elements import *
from uuid import uuid4
from connection_data import *


def test_graph_entity_snapshot():
    """
    Test insertion of data and then snapshot data 
    """
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
 
    event = Element(id = str(uuid4()))
    event.traits = ["Event"]
    event.activity = ["[2024-01-01T00:00:00;2025-01-01T00:00:00["]
    event.add_attribute_value("name", "A good event")
    event.add_attribute_value("state", "closed", ["[2024-01-01T00:00:00;2024-02-01T00:00:00["])
    event.add_attribute_value("state", "open", ["[2024-02-01T00:00:00;2025-01-01T00:00:00["])
    upsert_element_in_graph(token, graph_id, event)
 
    other_event = Element(id = str(uuid4()))
    other_event.traits = ["Event"]
    other_event.activity = ["[2020-01-01T00:00:00;2021-01-01T00:00:00["]
    other_event.add_attribute_value("name", "A small event")
    upsert_element_in_graph(token, graph_id, other_event)

    # load at a given moment 
    moment = datetime(2024, 5, 1)
    result = load_snapshot_graph_by_id(token, graph_id, moment)

    expected_graph = Graph(graph_id, name = "first graph")
    # france is kept, no change but period    
    new_france = france.copy_values()
    new_france.clean_periods()
    expected_graph.add_node(new_france, graph_id, True)
    # new event is kept, but value for closed is not set
    new_event = event.copy_values()
    new_event.clean_periods()
    new_event.remove_attribute_value("state", "closed")
    expected_graph.add_node(new_event, graph_id, True)
    # no more other_event

    assert expected_graph == result
    # and of course, clean
    assert clear_all_graphs(token)


def test_graph_roles_snapshot():
    """
    Test insertion of data and then snapshot data 
    """
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

    spain = Element(id = str(uuid4()))
    spain.traits = ["Country"]
    spain.activity = ["]-oo;+oo["]
    spain.add_attribute_value("name", "Spain")
    upsert_element_in_graph(token, graph_id, spain)
 
    event = Element(id = str(uuid4()))
    event.traits = ["Event"]
    event.activity = ["[2024-01-01T00:00:00;2025-01-01T00:00:00["]
    # not setting property on purpose, to test if data comes back
    # event.add_attribute_value("name", "a good event")
    upsert_element_in_graph(token, graph_id, event)
 
    relation = Element(id = str(uuid4()))
    relation.traits = ["Agenda"]
    relation.activity = ["[2024-01-01T00:00:00;2025-01-01T00:00:00["]
    relation.add_role_value("subject", event.element_id, ["[2024-01-01T00:00:00;2025-01-01T00:00:00["])
    relation.add_role_value("object", spain.element_id, ["[2024-01-01T00:00:00;2024-04-01T00:00:00["])
    relation.add_role_value("object", france.element_id, ["[2024-04-01T00:00:00;2025-01-01T00:00:00["])
    upsert_element_in_graph(token, graph_id, relation)

    # load at a given moment 
    moment = datetime(2024, 5, 1)
    result = load_snapshot_graph_by_id(token, graph_id, moment)

    expected_graph = Graph(graph_id, name = "first graph")
    # france is kept, no change but period    
    new_france = france.copy_values()
    new_france.clean_periods()
    expected_graph.add_node(new_france, graph_id, True)
    # spain too
    new_spain = spain.copy_values()
    new_spain.clean_periods()
    expected_graph.add_node(new_spain, graph_id, True)
    # new event is kept, but value for closed is not set
    new_event = event.copy_values()
    new_event.clean_periods()
    expected_graph.add_node(new_event, graph_id, True)
    # relation is kept but role of object is spain only 
    new_relation = relation.copy_values()
    new_relation.clean_periods()
    new_relation.remove_role_value("object", spain.element_id)
    expected_graph.add_node(new_relation, graph_id, True)

    
    assert expected_graph == result
    # and of course, clean
    assert clear_all_graphs(token)