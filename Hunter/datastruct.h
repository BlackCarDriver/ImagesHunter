#ifndef DATASTRUCT_H
#define DATASTRUCT_H
#include <iostream>
#include <QDialog>
#include <string.h>
#include <string>
using namespace std;

/*
 DataStruct is a pure virtual class to produce other class that
 translate different type of data between qt and go
*/
class DataStruct {
public:
    DataStruct();
    virtual ~DataStruct();
    virtual QString toString()=0;
    virtual void init(QString)=0;
};

//====================== SimpleStr =================

class simpleStr : public DataStruct {
public:
    QString data;
public:
    ~simpleStr();
    QString toString();
    void init(QString jstr);
};


#endif // DATASTRUCT_H
