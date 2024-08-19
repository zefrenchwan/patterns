from api import *
from elements import *
from uuid import uuid4
from connection_data import *

def test_element_deletion():
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

    # test if all elements inserted
    source_graph = load_graph_by_id(token, graph_id)
    assert len(source_graph.elements) == 3, "expected 3 elements, got " + str(len(source_graph.elements)) 
   

    # cannot delete paris and france, because they are linked to relation capitale
    assert not delete_element(token, paris.element_id) 
    assert not delete_element(token, france.element_id)
    assert delete_element(token, capitale.element_id)
    # once there is no relation to block, all elements may be deleted
    assert delete_element(token, paris.element_id) 
    assert delete_element(token, france.element_id)
    
    assert clear_all_graphs(token)


def test_element_attributes_loads():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    graph_id = create_graph(token,"first graph")
    assert graph_id is not None, "failed to create graph"

    element = Element(str(uuid4()))
    now_value = datetime_to_api_string(datetime.now())
    later_value = datetime_to_api_string(datetime.now() + timedelta(days = 35.0))
    element.activity = ["[" + now_value + ";+oo["]
    
    # add one attribute value then the other 
    element.add_attribute_value("name", "value", {"[" + now_value +";" + later_value + "]"})
    upsert_element_in_graph(token, graph_id, element)
    element.add_attribute_value("name", "other value", {"]" + later_value + ";+oo[" })
    upsert_element_in_graph(token, graph_id, element)
    source_graph = load_graph_by_id(token, graph_id)
    assert len(source_graph.elements) == 1, "insertion failure"
    loaded = source_graph.elements[element.element_id].element
    assert loaded == element, "loaded and stored differ: got " + str(loaded) + " and expected " + str(element)

    # add another value with a different name
    element.add_attribute_value("other name", "final value", {"[" + now_value +";" + later_value + "]"})
    upsert_element_in_graph(token, graph_id, element)
    source_graph = load_graph_by_id(token, graph_id)
    assert len(source_graph.elements) == 1, "insertion failure"
    loaded = source_graph.elements[element.element_id].element
    assert loaded == element, "loaded and stored differ: got " + str(loaded) + " and expected " + str(element)

    assert clear_all_graphs(token)


def test_elements_relations_load():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    graph_id = create_graph(token,"first graph")
    assert graph_id is not None, "failed to create graph"


    good_ids = []
    bad_ids = []
    for i in range(0,5):
        new_id = str(uuid4())
        source = Element(new_id)
        source.traits = ['A good person']
        source.activity = ["]-oo;+oo["]
        source.add_attribute_value("index", str(i))
        good_ids.append(new_id)
        upsert_element_in_graph(token, graph_id, source)
    source_graph = load_graph_by_id(token, graph_id)
    assert len(source_graph.elements) == len(good_ids), "missing entities from first insert"

    for i in range(0,5):
        new_id = str(uuid4())
        source = Element(new_id)
        source.traits = ['A bad person']
        source.activity = ["]-oo;+oo["]
        source.add_attribute_value("value", str(i))
        bad_ids.append(new_id)
        upsert_element_in_graph(token, graph_id, source)
    source_graph = load_graph_by_id(token, graph_id)
    assert len(source_graph.elements) == len(good_ids) + len(bad_ids), "missing entities from second insert"
    
    relation = Element(str(uuid4()))
    relation.traits = ['The whole together']
    relation.activity = ["]-oo;+oo["]
    for current_id in good_ids:
        relation.add_role_value("good", current_id, ["[2024-01-01T00:00:00;+oo["])
    for current_id in bad_ids:
        relation.add_role_value("bad", current_id)
    upsert_element_in_graph(token, graph_id, relation)

    source_graph = load_graph_by_id(token, graph_id)
    expected_size = len(good_ids) + len(bad_ids) + 1
    assert len(source_graph.elements) == expected_size, "insertion failure, expected " + str(expected_size) + " got " + str(len(source_graph.elements))
    loaded = source_graph.elements[relation.element_id].element
    assert loaded is not None, "missing relation"
    
    assert loaded == relation, "expected: " + str(relation.__dict__) + " got: " + str(loaded.__dict__)

    assert clear_all_graphs(token)


def test_dynamic_import_graph():
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
 
    # create a new graph and then import first graph
    new_graph_id = create_graph(token,"second graph")
    assert new_graph_id is not None, "failed to create graph"
    validation = add_imported_graph_to_current_graph(token, new_graph_id, graph_id)
    assert validation, "did not add graph import"
    # when reloaded, data from first graph is visible
    source_graph = load_graph_by_id(token, new_graph_id)
    assert len(source_graph.elements) == 1

    # test if cycles are detected
    validation = add_imported_graph_to_current_graph(token, graph_id, new_graph_id)
    assert not validation, "expected to find cycle, did not find it"
    

    assert clear_all_graphs(token)
