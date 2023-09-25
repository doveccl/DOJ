FROM python
WORKDIR /src
COPY source main.py
RUN python -m compileall main.py
RUN mv __pycache__/*.pyc main.pyc && rm -rf *.py __pycache__
CMD [ "python", "main.pyc" ]
