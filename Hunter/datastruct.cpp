#include "datastruct.h"
#include <QJsonObject>
#include <QJsonDocument>

//===================== DataStruct ===============

DataStruct::DataStruct(){}

DataStruct::~DataStruct(){}

//====================== SimpleStr =================



simpleStr::~simpleStr(){}

QString simpleStr::toString(){
    return data;
}

void simpleStr::init(QString jstr){
    this->data= jstr;
}
