# Patterns

The goal of this project is to:
* store data with no previous model definition
* store data that depends on time
* search information based on time

## Concepts

This project deals with data and metadata:
* data is elements linked together
* metadata is data about those elements. **Traits** defines types of elements

Let us give a preminilary example:
* metadata is "City", "Country" as traits. 
* Paris is the name of a data object, its trait is "City"
* Same idea, France is a data object, its traits is "Country"
* let us add metadata "Capital City" as another trait
* Capitale(Paris, France) is a data object (a relation, to be precise), its trait is "Capital City"

### Data 

Each data element is either an entity or a relation:
* Paris, France are entities
* CapitalCity(Paris, France) is a relation

Both contain:
* an **id** 
* an **activity** that defines the life cycle of the elemnt. For instance, an human life. 

For an entity:
* **attributes** that are a name and time dependant values

For a relation:
* **roles** and **values** as a map. For instance: subject = Paris, Object = Europe
* values are time-dependent: they may appear in a relation during a given period, not the full relation lifecycle

### Metadata

Metadata is represented as **traits** to define types of elements. 
A relation has traits too. 
A trait is not a simple label to put on elements. 



## Architecture

This project is a webapp not following REST standard. 

Its storage is currently a relational database (postgresql indeed). 

The project contains:
* **nodes** that defines the data model based on nodes in graphs
* **graphs** that defines the graph data model based on nodes
* **storage** that contains the storage system
* **serving** that contains the webapp part

## Installation

### Prerequisites

* **Go** version 1.22 or higher
* **Postgresql** installed and accessible

### Procedure

1. `go build` to build the application 
2. launch scripts in `storage/sql`. Execute sql data definition then procedures creations
3. define `PATTERNS_PORT` as the port to open to access the api, and `PATTERNS_DB_URL` to connect the database (postgresql)
4. launch go built application

### Create first users

Use procedures to insert users. 

For instance (pay attention to password, change it): 
* call susers.insert_user('root','password so secret that no one would find it');
* call susers.insert_super_user_roles('root'); 


## Testing 

1. Some unit tests, in packages with a `_test` suffix. It validates basic and local behavior
2. Some end to end tests. Assuming the api is up, database is up, python code launches tests. This code is located in `tests` folder