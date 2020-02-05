#include "bridge.h"
#include "datastruct.h"
#include<QDebug>
#include<QMessageBox>

using namespace std;


Bridge::Bridge(QWidget *parent):QWidget(parent){
    tcpServer = new QTcpServer();
    tcpSocket = new QTcpSocket();
    readDataLock = false;
    readDataBuff = "";
    Isconnected = false;
}

//监听端口并设置等待连接信号
int Bridge::start(){
    if(!tcpServer->listen(QHostAddress::LocalHost, ListenAt)) {
        return -1;
    }
    connect(tcpServer, SIGNAL(newConnection()), this, SLOT(MakeSocketConnection()));
    return 0;
}

//连接成功后升级连接方式为socket,设置收到消息和连接断开触发的事件
void Bridge::MakeSocketConnection(){
    qDebug()<<"Someone connect!"<<endl;
    tcpSocket = tcpServer->nextPendingConnection();
    if(!tcpSocket){
           QMessageBox::warning(this, "Msg", "Connect Fail!");
    } else {
           Isconnected = true;
           emit sendSignal("connect_success");
           connect(tcpSocket, SIGNAL(readyRead()), this, SLOT(SocketReadData()));
           connect(tcpSocket, SIGNAL(disconnected()), this, SLOT(SocketDisconect()));
    }
    return;
}

//处理量连接断开的情况
void Bridge::SocketDisconect(){
    QMessageBox::warning(this,"Inof","The connect is Close!");
    this->Isconnected = false;
    tcpSocket->close();
    tcpServer->close();
    emit sendSignal("disconnect");
}

//处理接收到的字符串msg,根据协议得到并写进key和contnet中。
bool Bridge::handleMsg(QString msg, QString &key, QString &content){
    int idx = msg.indexOf("@");
    if (idx < 0 ){
        qDebug()<<"Error: no @ in receive data!";
        return false;
    }
    key = msg.left(idx);
    content = msg.right(msg.length()-idx-1);
    content.replace("^#", "#");
    if (content.endsWith("\\#")){
        content = content.left(content.length()-2);
    }
    return true;
}

//disconnect initiative
void Bridge::disconnect(){
    this->SocketDisconect();
}

//send data to client
//key defined the function and can't contain char '@'
void Bridge::sendMessage(string key, DataStruct *data){
    if(key.find("@")!=key.npos){
        QMessageBox::warning(this, "Error", "Unexpect key!");
        return;
    }
    QString content = QString(data->toString());
    content.replace("#", "^#");
    string package = key + "@" + content.toStdString() + "\\#";
    if(int(package.length()*sizeof(char))>this->MTU){
        QMessageBox::warning(this, "Error", "Sending data overflow!");
        return;
    }
    qDebug()<<package.c_str();
    tcpSocket->write(package.c_str(), sizeof(char)*package.length());
}

//read data from socket connection
void Bridge::SocketReadData(){
    while(readDataLock);
    readDataLock = true;
    char buffer[1025];
    QString qs = "";
    QString key="", content = "";

    long long res = tcpSocket->read(buffer, 1024);
    qDebug()<<res;
    if (res==-1){
        qDebug()<<"Read data fail, return -1!";
        readDataLock = false;
        return;
    }

    qs = buffer;
    if(!qs.contains("\\#")){    //read data not completly
      qDebug()<<"qs not contain \\#! :"<<qs;
      readDataBuff += qs;
      readDataLock = false;
      return;
    }

    if(readDataBuff!="") {
        qDebug()<<"qs add with buff!";
        qs = readDataBuff + qs;
        readDataBuff = "";
    }


    //read those complete message
    QStringList  msgList = qs.split("\\#");
    for(int i=0;i<msgList.length()-1; i++){
        bool canRead = handleMsg(msgList[i], key, content);
        if (!canRead) continue;
        emit getMsg(key, content);
    }
    //ignore if last message not complete
    if (qs.endsWith("\\#")){
        bool canRead = handleMsg(msgList[msgList.length()-1], key, content);
        if (canRead) {
           emit getMsg(key, content);
        }
    }

    readDataLock = false;
    return;
}



