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

//监听端口,设置等待连接信号,启动go程序
int Bridge::start(){
    if(!tcpServer->listen(QHostAddress::LocalHost, ListenAt)) {
        return -1;
    }
    connect(tcpServer, SIGNAL(newConnection()), this, SLOT(MakeSocketConnection()));
    qDebug()<<QDir::currentPath();
    QProcess *pro = new QProcess(Q_NULLPTR);
    pro->start("./main.exe");
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

//主动断开连接
void Bridge::disconnect(){
    this->SocketDisconect();
}

//处理量连接被动断开的情况
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

//发送数据至服务端
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

//读取并处理从服务端收到的数据
void Bridge::SocketReadData(){
    while(readDataLock);
    readDataLock = true;
    char buffer[10241];
    QString qs = "", key="", content = "";
    long long res = tcpSocket->read(buffer, 10240);
    qDebug()<<res;
    if (res==-1){
        qDebug()<<"Read data fail, return -1!";
        readDataLock = false;
        return;
    }
    qs = buffer;
    //若接收到的消息不完整，放到缓存中等待下次处理
    if(!qs.contains("\\#")){
      qDebug()<<"qs not contain \\#! :"<<qs;
      readDataBuff += qs;
      readDataLock = false;
      return;
    }
    //若缓存非空则拼接到上次未处理的缓存
    if(readDataBuff!="") {
        qDebug()<<"qs add with buff!";
        qs = readDataBuff + qs;
        readDataBuff = "";
    }

    //QStringList 是完整的消息列表，以'\\#'结尾标准着消息完整，
    QStringList  msgList = qs.split("\\#");
    //先处理已经接受完整的消息
    for(int i=0;i<msgList.length()-1; i++){
        bool canRead = handleMsg(msgList[i], key, content);
        if (!canRead) continue;
        execHandle(key, content);
    }
    //末尾不完整部分视为错误数据暂时丢弃处理,否则照常处理
    if (qs.endsWith("\\#")){
        bool canRead = handleMsg(msgList[msgList.length()-1], key, content);
        if (canRead) {
           execHandle(key, content);
        }
    }
    readDataLock = false;
    return;
}

//=========================== Handle Class ============

//为target_class赋值
int Bridge::regisitClaas(void *classP){
    if(classP==nullptr){
        return -1;
    }
    if(target_class!=nullptr){
        return -1;
    }
    target_class = classP;
    return 0;
}

//注册消息处理函数，
int Bridge::regisitFunc(QString keyword, funcTypeP func){
    if(keyword.length()<3 || keyword.contains(" ") || func==nullptr ){
        qDebug()<<"[bridge.cpp => regisitFunc()]： argument illeagle";
        return -1;
    }
    if(funcMap.find(keyword)!=funcMap.end()){
        qDebug()<<"[bridge.cpp => regisitFunc()]： keyword already regisited, keyword="+keyword;
        return -1;
    }
    funcMap[keyword] = func;
    return 0;
}

//将收到的数据转交给对应的消息处理成函数进行处理，成功返回0
int Bridge::execHandle(QString keyword, QString content){
    if(keyword.length()<2 || keyword.contains(" ") || content=="" ){
        qDebug()<<"[bridge.cpp => execHandle()]： argument illeagle";
        return -1;
    }
    if(funcMap.find(keyword)==funcMap.end()){
        qDebug()<<"[bridge.cpp => execHandle()]： no such key, keyword="+keyword;
        return -1;
    }
    funcTypeP func = funcMap[keyword];
    int res = (*func)( target_class, content);
    if (res < 0){
        qDebug()<<"[bridge.cpp => execHandle()]：handle func run fail, keyword="+keyword;
        return -1;
    }
    return 0;
}
