test_cases = [
    {
        "id": "ut_HandleNonAUTHCmd_001",
        "name": """
        [Given] a password is set in the configuration
		[When] a non-AUTH command is received
            [And] the connection bearing the command is not authenticated
        [Then] returns error stating the connection requires authentication
        """,
        "test": {
            "given_config": {"password": '"justapass"'},
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [{"req": {"get": ("foo")}, "respError": True}],
            "then_data": {"src": [], "dst": []},
        },
    },
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
    },
    {
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
    },
    {
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
    },
    {
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
    },
    {
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
    },
    {
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
    },
    # === HandlePING ===
    {
        "id": "ut_HandlePING_001",
        "name": """
        """,
        "test": {
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [{"req": {"ping": ()}, "resp": True}],
            "then_data": {"src": [], "dst": []},
        },
    },
    # === HandleDefault ===
    {
        "id": "ut_HandleDefault",
        "name": """
        """,
        "test": {
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [
                {"req": {"echo": ("Hello")}, "resp": b"Hello"},
                {"req": {"hget": ("myhash", "field")}, "resp": None},
                {"req": {"hset": ()}, "respError": True},
                {"req": {"ttl": ("mykey")}, "resp": -2},
                # {"req": {"command": ()}, "resp": ("GET", "SET")},
            ],
            "then_data": {"src": [], "dst": []},
        },
    },
    # === HandleAUTH ===
    {
        "id": "ut_HandleAUTH_001",
        "name": """
        [Given] a Password is set in configuration
        [When] an AUTH command is received
            [And] the password argument matches with the one set in config
        [Then] returns OK
            [And] authenticate the connection
        """,
        "test": {
            "given_config": {"password": '"justapass"'},
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [{"req": {"auth": ("justapass")}, "resp": True}],
            "then_data": {"src": [], "dst": []},
        },
    },
    {
        "id": "ut_HandleAUTH_002",
        "name": """
        [Given] a Password is set in configuration
        [When] an AUTH command is received
            [And] the password argument doesn't match with the one set in config
        [Then] returns error stating invalid password
        """,
        "test": {
            "given_config": {"password": '"justapass"'},
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [{"req": {"auth": ("wrongpass")}, "respError": True}],
            "then_data": {"src": [], "dst": []},
        },
    },
    {
        "id": "ut_HandleAUTH_003",
        "name": """
        [Given] a password is not set in the configuration
        [When] an AUTH command is received
        [Then] returns error stating that password is not set
        """,
        "test": {
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [
                {"req": {"auth": ("nonexistent")}, "respError": True}
            ],
            "then_data": {"src": [], "dst": []},
        },
    },
    {
        "id": "ut_HandleAUTH_004",
        "name": """
        [When] an incorrect AUTH command is received (wrong number of args)
        [Then] returns error stating that the number of args is wrong
        """,
        "test": {
            "given_config": {"password": '"justapass"'},
            "given_data": {"src": [], "dst": []},
            "when_req_then_resp": [{"req": {"auth": ()}, "respError": True}],
            "then_data": {"src": [], "dst": []},
        },
    },
]
