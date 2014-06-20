from sqlalchemy import (Column, String, Binary, create_engine, Enum, Boolean, Integer)
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker, relationship

path = "sqlite:///dat.db"
engine = create_engine(path, echo=True)
Base = declarative_base()
Session = sessionmaker(bind=engine)

class Tx(Base):
    __tablename__ = "transactions"

    txid     = Column(String, primary_key=True)
    block    = Column(String)
    raw      = Column(Binary)

class Block(Base):
    __tablename__ = "blocks"

    hash     = Column(String, primary_key=True)
    prevhash = Column(String)

if __name__ == "__main__":
    Base.metadata.create_all(engine)
