#ifndef BRIDGE_H
#define BRIDGE_H
#include "datastruct.h"
#include <map>
#include <QDialog>
#include <QTcpServer>
#include <QtNetwork>
#include <QMainWindow>

using namespace std;

template <class T>
class Bridge : public QWidget{
    Q_OBJECT
    T *OBJ_class;                       //目标类
    typedef int(T::*funcTypeP)(QString);	//目标函数指针类型
    map<QString, funcTypeP>funcMap;		//关键字到目标函数的映射

public:
    Bridge(QWidget *parent = nullptr);
    virtual ~Bridge(){}
    int regisitClass(T *obj);           //设置目标类成员
    int regisitFunc(QString, funcTypeP); //注册处理函数
    int start();
    void disconnect();
    void sendMessage(string key, DataStruct *data);
    int execHandle(QString key, QString content);
    bool Isconnected;

signals:
      void getMsg(QString key, QString conetent);
      void sendSignal(QString sig);

private slots:
     void MakeSocketConnection();
     void SocketReadData();
     void SocketDisconect();

private:
    QTcpServer *tcpServer;
    QTcpSocket *tcpSocket;
    bool readDataLock;
    QString readDataBuff;
    bool handleMsg(QString msg,QString &key, QString &content);
    const quint16 ListenAt = 4747;
    const int MTU = 1024 * 100;
};

#endif // BRIDGE_H
