.PHONY: setup-virtual-env 
setup-virtual-env:
	python3 -m venv remiro-e2e-env
	. remiro-e2e-env/bin/activate
	pip install -r requirements.txt

.PHONY: e2e-test
e2e-test: setup-virtual-env
	python e2e.py

.DEFAULT_GOAL := e2e-test