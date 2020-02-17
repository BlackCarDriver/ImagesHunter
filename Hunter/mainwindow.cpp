#include "mainwindow.h"
#include "ui_mainwindow.h"
#include "bridge.h"
#include <QMessageBox>
#include<QMenu>
#include <QFileDialog>
using namespace  std;

MainWindow::MainWindow(QWidget *parent):QMainWindow(parent) ,ui(new Ui::MainWindow){
    setWindowIcon(QIcon(":./icom.ico"));
    ui->setupUi(this);
    ui->static_list->horizontalHeader()->setStretchLastSection(true);

    //初始化bridge
    bridge = new Bridge();
    bridge->regisitClaas(this);
    //等待socket连接
    int suc = bridge->start();
    if (suc>=0){
        //通过下面这个信号槽来实现在bridge中对mainwindows进行控制
       connect(bridge, SIGNAL(sendSignal(QString)), this, SLOT(functionHandle(QString)));
    }else{
       QMessageBox::warning(this, "Error", "Fail when listen at localhost:4747!");
    }

    //菜单栏相关设置
    QMenu* file = menuBar()->addMenu(tr("文件(&F)"));
    QAction *openconf = new QAction(tr("打开配置(&O)"), this);
    QAction *saveconf = new QAction(tr("保存配置(&S)"), this);
    QAction *quit = new QAction(tr("退出(&Q)"), this);
    file->addAction(openconf);
    file->addAction(saveconf);
    file->addAction(quit);
    QMenu* help = menuBar()->addMenu(tr("帮助(&H)"));
    QAction *connect = new QAction(tr("联系作者(&C)"), this);
    QAction *seehelp = new QAction(tr("查看帮助(&S)"), this);
    QAction *seeversion = new QAction(tr("版本信息(&V)"), this);
    help->addAction(seehelp);
    help->addAction(connect);
    help->addAction(seeversion);
    QMenu* lanugh = menuBar()->addMenu(tr("语言(&L)"));
    QAction *chinese = new QAction(tr("中文(&C)"), this);
    QAction *english = new QAction(tr("&English"), this);
    lanugh-> addAction(english);
    lanugh->addAction(chinese);

    //在连接成功前一些按钮设置为无效状态
    widgetArray.push_back(ui->btn_start);
    widgetArray.push_back(ui->btn_stop);
    setWidgetState(false);

    //注册消息处理函数
    bridge->regisitFunc("debug", debugHandle);
    bridge->regisitFunc("error", errorHandle);
    bridge->regisitFunc("info", infoHandle);
    bridge->regisitFunc("table", tableHandle);
    bridge->regisitFunc("static", staticHandle);
    bridge->regisitFunc("function", controlWindow);
}

MainWindow::~MainWindow(){
    delete ui;
}

//改变指定组件的状态，设置是否可点击
void MainWindow::setWidgetState(bool enable){
    for(uint i=0;i < widgetArray.size();i++){
        widgetArray[i]->setEnabled(enable);
    }
    return;
}

//提供接口给bridge类来控制mainwindows
void MainWindow::functionHandle(QString key){
    qDebug()<<"function key: "<<key;
    if(key=="connect_success"){ //成功与go端建立连接
        this->setWidgetState(true);
        ui->btn_stop->setEnabled(false);
    }
    if(key=="disconnect"){  //go端主动结束或强制结束
        ui->static_progressbar->setValue(100);  //进度条设置成100%
        QMessageBox::warning(this,"Inof","The connect is Close!");
        ui->btn_stop->setEnabled(false);
        ui->btn_start->setEnabled(false);
        ui->btn_start->setText("开始");
    }
    return;
}

/*
从组件中获取用户设置的值,通过返回值返回
含义与规则: "%s %s %d %d %d %d %d %d %d %s %s %s %d %d",
&method,
&savePath,
&totalSizeLimit,
&numberLimit,
&threadLimit,
&minSizeLimit,
&maxSizeLimit,
&waitTimeLimit,
&intervalTime,
&baseUrl,
&linkKey,
&targetKey,
&startPoint,
&endPoint
特殊串特殊处理:空字符串转“&empty”, 空格转“&space”
*/
QString MainWindow::getConfig(){
      //公用配置值
      int sizeLimit = ui->spin_base_sizeLimit->value();
      int numberLimit = ui->spin_base_numLimit->value();
      int threadLimit = ui->spin_base_threadLimit->value();
      int minmun = ui->spin_base_mininum->value();
      int maxnum = ui->spin_base_maxnum->value();
      int longestWait = ui->spin_base_waitTime->value();
      int interval = ui->spin_base_interval->value();
      QString savePath = ui->edit_base_savePath->text().replace(" ", "&space");
      //方案配置值
      QString method ="&empty", baseUrl="&empty", lineKey="&empty", targetKey="&empty";
      int startPoint =0, endPoint=0;
      int tabSelect = ui->methodTag->currentIndex();
      //根据方案不同对特定配置进行处理
      switch (tabSelect) {
        case 0: //bfs
          method = "BFS";
          baseUrl = ( ui->edit_bfs_url->text()==""?"&empty":ui->edit_bfs_url->text());
          lineKey = (ui->edit_bfs_lineKey->text()==""?"&empty":ui->edit_bfs_lineKey->text());
          targetKey = (ui->edit_bfs_targetKey->text()==""?"&empty":ui->edit_bfs_targetKey->text());
          break;
        case 1: //dfs
          method = "DFS";
          baseUrl = (ui->edit_dfs_url->text()==""?"&empty":ui->edit_dfs_url->text());
          lineKey = (ui->edit_dfs_lineKey->text()==""?"&empty":ui->edit_dfs_lineKey->text());
          targetKey = (ui->edit_dfs_targetKey->text()==""?"&empty":ui->edit_dfs_targetKey->text());
          break;
        case 2: //for loop
          method = "FOR";
          baseUrl = (ui->edit_for_url->text()==""?"&empty":ui->edit_for_url->text());
          targetKey = (ui->edit_for_targetKey->text()==""?"&empty":ui->edit_for_targetKey->text());
          break;
        case 3: //list
          method = "LIST";
          baseUrl = (ui->textEdit_urlList->placeholderText()==""?"&empty":ui->textEdit_urlList->placeholderText());
          break;
      }
      //检查必填选项
      if(savePath==""){
          QMessageBox::warning(this,"Fail","SavePath is null!");
          return "";
      }
      if(baseUrl=="&empty"){
          QMessageBox::warning(this,"Fail","URL format is invalid!");
          return "";
      }
      //检查纠正和加工
      method = method.replace(" ", "&space");
      baseUrl = baseUrl.replace(" ", "&space");
      lineKey = lineKey.replace(" ", "&space");
      targetKey = targetKey.replace(" ", "&space");

      QString configStr =  QString("%1 %2 %3 %4 %5 %6 %7 %8 %9 %10 %11 %12 %13 %14\
").arg(method).arg(savePath).arg(sizeLimit).arg(numberLimit).arg(threadLimit).arg(minmun).arg(maxnum).arg(longestWait).arg(interval).arg(baseUrl).arg(lineKey).arg(targetKey).arg(startPoint).arg(endPoint);
      return configStr;
}


//========================= 按钮点击事件 =========================

//开始/暂停按钮点击事件
void MainWindow::on_btn_start_clicked(){
    simpleStr *conf = new simpleStr;
    QString state = ui->btn_start->text();
    if (state=="开始"){
        ui->static_list->clear();//清空列表
    }
    if(state=="暂停"){
//      ui->btn_stop->setEnabled(false);
        conf->init("...");
        bridge->sendMessage("pause", conf);
        ui->btn_start->setText("继续");
    }else if(state=="开始" || state=="继续") {
        ui->btn_stop->setEnabled(true);
        conf->init(getConfig());
        bridge->sendMessage("start", conf);
        qDebug()<<conf->toString();
        ui->btn_start->setText("暂停");
    }
    delete conf;
    return;
}

//停止按钮点击事件
void MainWindow::on_btn_stop_clicked(){
    simpleStr *conf = new simpleStr;
    conf->init("....");
    bridge->sendMessage("stop", conf);
    ui->btn_stop->setEnabled(false);
    ui->btn_start->setEnabled(true);
    ui->btn_start->setText("开始");
    delete conf;
    return;
}

//选择路径按钮被点击事件
void MainWindow::on_pushButton_clicked(){
   QString dir = QFileDialog::getExistingDirectory( this, "保存位置", "D://",  QFileDialog::DontResolveSymlinks);
   ui->edit_base_savePath->setText(dir);
   return;
}


//========================= 消息处理函数 =========================

//debug消息处理
int MainWindow::debugHandle(void *thisP, QString content){
    MainWindow *This = static_cast<MainWindow*>(thisP);
    qDebug()<<"[mainwindows.cpp -> debugHandle()] content="<<content;
    QMessageBox::information(This, "debug", "content="+content);
    return 0;
}

//报错消息处理
int MainWindow::errorHandle(void *thisP, QString content){
    MainWindow *This = static_cast<MainWindow*>(thisP);
    qDebug()<<"[mainwindows.cpp -> errorHandle()] message="<<content;
    QMessageBox::information(This, "Error", "Receive a error from go:"+content);
    return 0;
}

//普通消息处理
int MainWindow::infoHandle(void *thisP, QString content){
    MainWindow *This = static_cast<MainWindow*>(thisP);
    qDebug()<<"[mainwindows.cpp -> infoHandle()] message="<<content;
    QMessageBox::information(This, "Error", "Receive a info from go:"+content);
    return 0;
}

//将收到的数据插入到下载报告列表里面的新行
int MainWindow::tableHandle(void *thisP, QString content){
    MainWindow *This = static_cast<MainWindow*>(thisP);
    QStringList res = content.replace("\\ ", " ").split(' ');
    if(res.length()<4){
        qDebug()<<"length of table data smaller than 4!";
        return -1;
    }
    int i = This->ui->static_list->rowCount();
    This->ui->static_list->insertRow(i);
    This->ui->static_list->setItem(i, 0, new QTableWidgetItem(res[0]));
    This->ui->static_list->setItem(i, 1, new QTableWidgetItem(res[1]));
    This->ui->static_list->setItem(i, 2, new QTableWidgetItem(res[2]));
    This->ui->static_list->setItem(i, 3, new QTableWidgetItem(res[3]));
    return 0;
}

/*
更新统计信息（总耗时， 总大小等...）
content规则："static@%d %d %d %d %dKB/s %s %d",
totalNumber,
totalBytes,
foundPageList.Len(),
pageNumber,
speed,
totalBytes,
percentage
*/
int MainWindow::staticHandle(void *thisP, QString content){
    MainWindow *This = static_cast<MainWindow*>(thisP);
    QStringList res = content.split(' ');
    if (res.length()!=7){
        qDebug()<<"length of static data not correct!";
        return -1;
    }
    This->ui->lable_res_number->setText(res[0]);
    This->ui->lable_res_size->setText(res[1]);
    This->ui->lable_res_length->setText(res[2]);
    This->ui->lable_res_page->setText(res[3]);
    This->ui->label_res_speed->setText(res[4]);
    This->ui->lable_res_time->setText(res[5]);
    This->ui->static_progressbar->setValue(res[6].toInt());
    return 0;
}

//对组件进行控制的功能集合
int MainWindow::controlWindow(void *thisP, QString content){
    MainWindow *This = static_cast<MainWindow*>(thisP);
    if (content == "auto_stop") {   //任务到达结束条件，自动结束
        This->ui->btn_stop->setEnabled(false);
        This->ui->btn_start->setEnabled(true);
        This->ui->btn_start->setText("开始");
        QMessageBox::information(This, "info", "The missing is complete! :"+content);
        return 0;
    }
    return -1;
}
