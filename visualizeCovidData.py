import matplotlib.pyplot as plt
import matplotlib.dates
from matplotlib.dates import AutoDateFormatter, AutoDateLocator
import math
import datetime
import smtplib, ssl
from email.mime.multipart import MIMEMultipart 
from email.mime.text import MIMEText 
from email.mime.application import MIMEApplication
import datetime
dates_2020 = []
dates_2021 = []
dates_2022 = []
innlagte_2020 = []
innlagte_2021 = []
innlagte_2022 = []
currentdate = -1

smittetall_csv_filename = "data/CovidSmitteTall.csv"


with open("data/NasjonalCovidData.csv") as csv_file:
    for line in csv_file:
        if line == "dato,innlagte,respirator\n":
            continue
        data = line.strip().split(",")
        currentdate = data[0].split("T")[0]
        if '2020' in currentdate:
            dates_2020.append(currentdate)
            innlagte_2020.append(int(data[1]))
        elif '2021' in currentdate:
            dates_2021.append(currentdate)
            innlagte_2021.append(int(data[1]))
        elif '2022' in currentdate:
            dates_2022.append(currentdate)
            innlagte_2022.append(int(data[1]))




converted_dates_2020 = matplotlib.dates.datestr2num(dates_2020) # 366 dager i forskjell
converted_dates_2021 = matplotlib.dates.datestr2num(dates_2021)
converted_dates_2021 = [date-366 for date in converted_dates_2021]
converted_dates_2022 = matplotlib.dates.datestr2num(dates_2022)
converted_dates_2022 = [date-365*2+1 for date in converted_dates_2022]

plt.plot_date(converted_dates_2020, innlagte_2020, '-', label="innlagte 2020")
plt.plot_date(converted_dates_2021, innlagte_2021, '-', label="innlagte 2021")
plt.plot_date(converted_dates_2022, innlagte_2022, '-', label="innlagte 2022")


plt.xlabel("dato")
plt.ylabel("Antall innlagte i Norge")
plt.title("innlagte for covid per dag i 2020, 2021 og 2022")
plt.legend()

xtick_locator = AutoDateLocator()
xtick_formatter = AutoDateFormatter(xtick_locator)
plt.grid(axis='y')
date_today = datetime.datetime.today().date()
plt.ylim(bottom=0)

plt.savefig('data/'+str(date_today)+'_high_dpi.png', dpi=1200) #bbox_inches='tight'
plt.show()



def send_email():
    """Sends an email with the produced image and the csv data as attachments"""
    smtp_host = "smtp.gmail.com"
    smtp_port = "587"
    from_email = ""
    to_email = ""
    password = ""
    with open("config.txt", 'r') as file:
        file.readline()
        password = file.readline().strip().split(":")[-1]
    if len(password) != 16:
        raise Exception("Password is in wrong format for google app password")
    
    message = MIMEMultipart('mixed')
    message['From'] = '<{sender}>'.format(sender = from_email)
    message['To'] = to_email
    date_today = str(datetime.datetime.today().date())
    message['Subject'] = f'Sammenligning av covid innleggelser 2020 vs 2021 for: {date_today}'
    imagePath = 'data/'+date_today+'_high_dpi.png'
    try:
        with open(imagePath, 'rb') as img:
            p = MIMEApplication(img.read(),_subtype="png")
            p.add_header('Content-Disposition', "attachment; filename= %s" % date_today+'_high_dpi.png')
            message.attach(p)
    except Exception as e:
        raise Exception(e)

    csv_path = "data/NasjonalCovidData.csv"
    try:
        with open(csv_path, 'rb') as csvFile:
            p = MIMEApplication(csvFile.read(),_subtype="csv")
            p.add_header('Content-Disposition', "attachment; filename= %s" % date_today+'.csv')
            message.attach(p)
    except Exception as e:
        print(str(e))

    msg_full = message.as_string()
    context = ssl.create_default_context()

    with smtplib.SMTP(smtp_host, smtp_port) as server:
        server.ehlo()  
        server.starttls(context=context)
        server.ehlo()
        server.login(from_email, password)
        server.sendmail(from_email, from_email, msg_full)
        server.quit()

    print("email sent out successfully")

def sentToday(date):
    with open("data/last_sent.txt", "r") as last_sent:
        if last_sent.readline() == date:
            return True
        return False


# if currentdate == str(date_today) and not sentToday(str(date_today)):
#     print("sending email")
#     send_email()
#     with open("data/last_sent.txt", "w") as last_sent:
#         last_sent.write(str(date_today))
# else:
#     print("did not send email")