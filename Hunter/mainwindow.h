#ifndef MAINWINDOW_H
#define MAINWINDOW_H
#include "bridge.h"
#include <QMainWindow>
#include <vector>

QT_BEGIN_NAMESPACE
namespace Ui { class MainWindow; }
QT_END_NAMESPACE

class MainWindow : public QMainWindow{
    Q_OBJECT

public:
    MainWindow(QWidget *parent = nullptr);
    ~MainWindow();

private: //消息处理函数
    static int test(void *thisP, QString content);
    static int errorHandle(void *thisP, QString content);
    static int infoHandle(void *thisP, QString content);
    static int tableHandle(void *thisP, QString content);
    static int staticHandle(void *thisP, QString content);

private slots:
    void on_btn_start_clicked();
    void messageHandle(QString key, QString content);
    void functionHandle(QString key);

    void on_pushButton_clicked();

    void on_btn_stop_clicked();

private:
    Ui::MainWindow *ui;
    Bridge *bridge;
    vector<QWidget*> widgetArray;
    void setWidgetState(bool);
    QString getConfig();
};

#endif // MAINWINDOW_H

