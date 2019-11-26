#ifndef BRIDGE_H
#define BRIDGE_H
#include "datastruct.h"
#include <QDialog>
#include <QTcpServer>
#include <QtNetwork>
#include <QMainWindow>

using namespace std;

/*
 bridge is the tool to connect qt and go ;
*/
class Bridge : public QWidget{
     Q_OBJECT
public:
    Bridge(QWidget *parent = nullptr);
    virtual ~Bridge(){}
    int start();
    void disconnect();
    void sendMessage(string key, DataStruct *data);
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
