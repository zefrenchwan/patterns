from api import *
from elements import *
from uuid import uuid4
from connection_data import *


def test_graph_slices_snapshot():
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

    # test load to exclude event and relation 
    result = load_slice_graph_by_id(token, graph_id, datetime(2020,1,1), datetime(2021,1,1))
    expected_graph = Graph(graph_id, name = "first graph")
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(spain, graph_id, True)

    assert result == expected_graph
    
    # test when relation is valid to exclude one value
    result = load_slice_graph_by_id(token, graph_id, datetime(2024,6,1))
    expected_graph = Graph(graph_id, name = "first graph")
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(spain, graph_id, True)
    expected_graph.add_node(event, graph_id, True)
    # relation is kept but role of object is spain only 
    new_relation = relation.copy_values()
    new_relation.remove_role_value("object", spain.element_id)
    expected_graph.add_node(new_relation, graph_id, True)

    assert expected_graph == result

    # test after relation 
    result = load_slice_graph_by_id(token, graph_id, datetime(2030,1,1))
    expected_graph = Graph(graph_id, name = "first graph")
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(spain, graph_id, True)

    assert result == expected_graph


    # and of course, clean
    assert clear_all_graphs(token)