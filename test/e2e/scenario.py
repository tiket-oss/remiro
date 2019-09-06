test_cases = [
    {
        "id": "ut001",
        "name": """
        [Given] a key is available in "destination"
        [When] a GET request for the key is received
        [Then] GET and return the key value from "destination"
        """,
        "test": {
            # "given_config": {"delete_on_get": "true", "delete_on_set": "true"},
            "given_data": {"src": [], "dst": [{"set": ("foo", "bar")}]},
            "when_req_then_resp": [{"req": {"get": ("foo")}, "resp": b"bar"}],
            "then_data": {"src": [], "dst": [{"set": ("foo", "bar")}]},
        },
    },
    {
        "id": "ut002",
        "name": """
        [Given] a key is not available in "destination"
            [And] "destination" return non-nil error
        [When] a GET request for the key is received
        [Then] return the error from "destination"
        """,
        "test": {
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [{"req": {"get": ("foo")}, "resp": None}],
            "then_data": {"src": [], "dst": []},
        },
    },
    {
        "id": "ut004",
        "name": """
        [Given] a key is not available in "destination"
            [And] the key is available in "source"
            [And] deleteOnGet set to false
        [When] a GET request for the key is received
        [Then] GET and return the key value from "source"
            [And] SET the value with the key to "destination"
        """,
        "test": {
            "given_config": {"delete_on_get": "false"},
            "given_data": {"src": [{"set": ("foo", "bar")}], "dst": []},
            "when_req_then_resp": [{"req": {"get": ("foo")}, "resp": b"bar"}],
            "then_data": {
                "src": [{"set": ("foo", "bar")}],
                "dst": [{"set": ("foo", "bar")}],
            },
        },
    },
    {
        "id": "ut005",
        "name": """
        [Given] a key is not available in "destination"
            [And] the key is available in "source"
            [And] deleteOnGet set to true
        [When] a GET request for the key is received
        [Then] GET and return the key value from "source"
            [And] SET the value with the key to "destination"
            [And] DELETE the key from "source"
        """,
        "test": {
            "given_config": {"delete_on_get": "true"},
            "given_data": {"src": [{"set": ("foo", "bar")}], "dst": []},
            "when_req_then_resp": [{"req": {"get": ("foo")}, "resp": b"bar"}],
            "then_data": {"src": [], "dst": [{"set": ("foo", "bar")}]},
        },
    },
]
