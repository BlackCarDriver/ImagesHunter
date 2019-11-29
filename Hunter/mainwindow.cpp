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
    //setting up some widget
    ui->static_list->horizontalHeader()->setStretchLastSection(true);

    //start wait for socket connection
    bridge = new Bridge();
    int suc = bridge->start();
    if (suc>=0){
       connect(bridge, SIGNAL(getMsg(QString, QString)), this, SLOT(messageHandle(QString, QString)));
       connect(bridge, SIGNAL(sendSignal(QString)), this, SLOT(functionHandle(QString)));
    }else{
       QMessageBox::warning(this, "Error", "Fail when listen at localhost:4747!");
    }

    //menu bar set   ting
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

    //set some widget's state as disable before socket connect success
    widgetArray.push_back(ui->btn_start);
    widgetArray.push_back(ui->btn_stop);
   // setWidgetState(false);
}

MainWindow::~MainWindow(){
    delete ui;
    delete bridge;
}

//set widgets in widgetArray as enable or disable
void MainWindow::setWidgetState(bool enable){
    for(uint i=0;i < widgetArray.size();i++){
        widgetArray[i]->setEnabled(enable);
    }
    return;
}

//functionHandle is a interface for bridge to controll mainwindow
void MainWindow::functionHandle(QString key){
    qDebug()<<"function key: "<<key;
    if(key=="connect_success"){
        this->setWidgetState(true);
    }
    return;
}

//messageHandle handle the data or message get from go
void MainWindow::messageHandle(QString key, QString content){
    qDebug()<<"Message key:  "<<key;
    if(key=="error"){
        QMessageBox::information(this, "go error", content);
        return;
    }
    if(key=="test"){
        QStringList res = content.split(' ');
        if(res.length()<4){
            qDebug()<<"length of static data smaller than 4!";
            return;
        }
        int i = ui->static_list->rowCount();
        ui->static_list->insertRow(i);
        ui->static_list->setItem(i, 0, new QTableWidgetItem(res[0]));
        ui->static_list->setItem(i, 1, new QTableWidgetItem(res[1]));
        ui->static_list->setItem(i, 2, new QTableWidgetItem(res[2]));
        ui->static_list->setItem(i, 3, new QTableWidgetItem(res[3]));
        return;
    }
    return;
}

void MainWindow::on_btn_start_clicked(){
    simpleStr *conf = new simpleStr;
    conf->init(getConfig());
    bridge->sendMessage("start", conf);
    delete conf;
    return;
}

//collect those setting from widget to a string
QString MainWindow::getConfig(){
      //get basic config
      int sizeLimit = ui->spin_base_sizeLimit->value();
      int numberLimit = ui->spin_base_numLimit->value();
      int threadLimit = ui->spin_base_threadLimit->value();
      int minmun = ui->spin_base_mininum->value();
      int maxnum = ui->spin_base_maxnum->value();
      int longestWait = ui->spin_base_waitTime->value();
      int interval = ui->spin_base_interval->value();
      QString savePath = ui->edit_base_savePath->text();

      //get medhod config
      QString method ="-", baseUrl="-", lineKey="-", targetKey="-";
      int startPoint =0, endPoint=0;
      int tabSelect = ui->methodTag->currentIndex();
      switch (tabSelect) {
        case 0: //bfs
          method = "bfs";
          baseUrl = ( ui->edit_bfs_url->text()==""? "-":ui->edit_bfs_url->text());
          lineKey = (ui->edit_bfs_lineKey->text()==""?"-":ui->edit_bfs_url->text());
          targetKey = (ui->edit_bfs_targetKey->text()==""?"-":ui->edit_bfs_targetKey->text());
          break;
        case 1: //dfs
          method = "dfs";
          baseUrl = (ui->edit_dfs_url->text()==""?"-":ui->edit_dfs_url->text());
          lineKey = (ui->edit_dfs_lineKey->text()==""?"-":ui->edit_dfs_lineKey->text());
          targetKey = (ui->edit_dfs_targetKey->text()==""?"-":ui->edit_dfs_targetKey->text());
          break;
        case 2: //for loop
          method = "forloop";
          baseUrl = (ui->edit_for_url->text()==""?"-":ui->edit_for_url->text());
          targetKey = (ui->edit_for_targetKey->text()==""?"-":ui->edit_for_targetKey->text());
          break;
        case 3: //list
          method = "urllist";
          baseUrl = (ui->textEdit_urlList->placeholderText()==""?"-":ui->textEdit_urlList->placeholderText());
          break;
      }
      //check the value
      if(savePath==""){
          QMessageBox::warning(this,"Fail","SavePath is null!");
          return "";
      }
      if(baseUrl=="-"){
          QMessageBox::warning(this,"Fail","URL format is invalid!");
          return "";
      }
      QString configStr =  QString("%1 %2 %3 %4 %5 %6 %7 %8 %9 %10 %11 %12 %13 %14\
").arg(method).arg(savePath).arg(sizeLimit).arg(numberLimit).arg(threadLimit).arg(minmun).arg(maxnum).arg(longestWait).arg(interval).arg(baseUrl).arg(lineKey).arg(targetKey).arg(startPoint).arg(endPoint);
      qDebug()<<configStr;
      return configStr;
}

//choise a path to save the images
void MainWindow::on_pushButton_clicked(){
   QString dir = QFileDialog::getExistingDirectory( this, "保存位置", "D://",  QFileDialog::DontResolveSymlinks);
   ui->edit_base_savePath->setText(dir);
   return;
}
