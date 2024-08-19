class AuthorizedGraphDTO:
    """
    AuthorizedGraphDTO contains the metadata and roles for each graph
    """
    def __init__(self, graph_id: str, roles: list[str], name:str, description: str = "", metadata: dict[str,list[str]] = dict()):
        self.graph_id = graph_id
        self.roles = sorted(list(set(roles)))
        self.name = name
        self.description = description
        self.metadata = metadata 

    def __repr__(self):
        return str(self)
    
    def __str__(self) -> str:
        return "AuthorizedGraphDTO[" + str(self.__dict__) + "]"
    
    def to_json(self) -> dict:
        return {
            "id":self.graph_id,
            "name":self.name,
            "roles": self.roles,
            "description":self.description,
            "metadata": self.metadata
        }


def authorized_graphs_from_json(data: dict) -> AuthorizedGraphDTO:
    id = data["id"]
    name = data["name"]
    roles = data["roles"]
    description = data.get("description")
    if description is None:
        description = ""
    metadata = data.get("metadata")
    if metadata is None:
        metadata = dict()
    return AuthorizedGraphDTO(id, roles, name, description, metadata)