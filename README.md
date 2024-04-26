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


### Metadata

Metadata is represented as **traits** to define types of elements. 
A relation has traits too. 


A trait is not a simple label to put on elements. 
Traits are grouped into **inheritance graphs**. 
For instance, "City" is a "Location". 
Flagging Paris as a City should create a trait "Location" for Paris too.
For instance, "Couple" for a relation is a special case for "Knows". 
So, if you flag a relation with "Couple", it is expected to be aware it also is "Knows". 



## Architecture

This project is a webapp not following REST standard. 
It allows:
* to store data using **/upsert/elements/** endpoints
* to retrieve data depending on time using **/snapshot/entities/** family of endpoints
* to load data using **/load/elements/** endpoints

Linked project [bootstrapper](https://github.com/zefrenchwan/bootstrapper) allows to init the first values to store and build relations with. 

Its storage is currently a relational database (postgresql indeed). 

The project contains:
* **patterns** that defines the data model
* **storage** that contains the storage system
* **serving** that contains the webapp part