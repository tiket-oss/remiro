test_cases = [
    {
        "id": "ut001",
        "name": """
        [Given] a key is available in "destination"
        [When] a GET request for the key is received
        [Then] GET and return the key value from "destination
        """,
        "test": {
            # "given_config": {"delete_on_get": "true", "delete_on_set": "true"},
            "given_data": {"src": [], "dst": [{"set": ("foo", "bar")}]},
            "when_req_then_resp": [{"req": {"get": ("foo")}, "resp": b"bar"}],
            "then_data": {"src": [], "dst": [{"set": ("foo", "bar")}]},
        },
    }
]
