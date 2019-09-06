test_cases = [
    # === HandleGET ===
    {
        "id": "ut_HandleGET_001",
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
    }, {
        "id": "ut_HandleGET_002",
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
    }, {
        "id": "ut_HandleGET_004",
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
    }, {
        "id": "ut_HandleGET_005",
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
    },{
        "id": "ut_HandleGET_006",
        "name": """
        [Given] a key is not available in "destination"
            [And] the key is not available in "source"
        [When] a GET request for the key is received
        [Then] return nil rawMessage from "source"
        """,
        "test": {
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [{"req": {"get": ("foo")}, "resp": None}],
            "then_data": {"src": [], "dst": []},
        },
    },
    # === HandleSET ===
    {
        "id": "ut_HandleSET_001",
        "name": """
        [Given] deleteOnSet set to false
        [When] a SET request for a key is received
        [Then] SET the key with the value to "destination"
        """,
        "test": {
            "given_config": {"delete_on_set": "false"},
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [{"req": {"set": ("foo", "bar")}, "resp": True}],
            "then_data": {"src": [], "dst": [{"set": ("foo", "bar")}]},
        },
    },{
        "id": "ut_HandleSET_003",
        "name": """
        [Given] deleteOnSet set to true
        [When] a SET request for a key is received
        [Then] SET the key with the value to "destination"
            [And] DELETE the key from "source"
        """,
        "test": {
            "given_config": {"delete_on_set": "true"},
            "given_data": {"src": [{"set": ("foo", "bar")}], "dst": []},
            "when_req_then_resp": [{"req": {"set": ("foo", "bar")}, "resp": True}],
            "then_data": {"src": [], "dst": [{"set": ("foo", "bar")}]},
        },
    }
]
